package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

func TestLargeDirectoryBoundedWorkers(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 8000; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("f%05d.jpg", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	var active, peak atomic.Int32
	var matches atomic.Int64
	decoder := func(string) (int, int, bool, error) {
		n := active.Add(1)
		for {
			old := peak.Load()
			if n <= old || peak.CompareAndSwap(old, n) {
				break
			}
		}
		runtime.Gosched()
		defer active.Add(-1)
		return 640, 480, true, nil
	}
	stats, err := scanTree(context.Background(), []string{root}, dimensionQuery{axis: "width", over: true, value: 500}, scanOptions{workers: 8, decode: decoder}, func(imageData) error { matches.Add(1); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if stats.files != 8000 || stats.matched != 8000 || matches.Load() != 8000 {
		t.Fatalf("bad totals: %+v matches %d", stats, matches.Load())
	}
	if peak.Load() > 8 {
		t.Fatalf("peak concurrency exceeded: %d", peak.Load())
	}
}

func TestTreeRecursionAndFileRoot(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, "nested"), 0700)
	_ = os.WriteFile(filepath.Join(root, "one.png"), nil, 0600)
	_ = os.WriteFile(filepath.Join(root, "nested", "two.png"), nil, 0600)
	decode := func(string) (int, int, bool, error) { return 100, 200, true, nil }
	for _, tc := range []struct {
		roots     []string
		recursive bool
		want      int64
	}{
		{[]string{root}, false, 1}, {[]string{root}, true, 2}, {[]string{filepath.Join(root, "one.png")}, false, 1},
	} {
		stats, err := scanTree(context.Background(), tc.roots, dimensionQuery{axis: "height", over: false, value: 201}, scanOptions{workers: 2, recursive: tc.recursive, decode: decode}, func(imageData) error { return nil })
		if err != nil || stats.matched != tc.want {
			t.Fatalf("scan %+v = %+v / %v", tc, stats, err)
		}
	}
}

func TestSymlinkCycleIgnored(t *testing.T) {
	root := t.TempDir()
	_ = os.WriteFile(filepath.Join(root, "ok.jpg"), nil, 0600)
	if err := os.Symlink(root, filepath.Join(root, "cycle")); err != nil {
		t.Skip(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	stats, err := scanTree(ctx, []string{root}, dimensionQuery{axis: "width", over: true, value: 1}, scanOptions{workers: 2, recursive: true, decode: func(string) (int, int, bool, error) { return 10, 10, true, nil }}, func(imageData) error { return nil })
	if err != nil || stats.matched != 1 {
		t.Fatalf("symlink traversal result %+v %v", stats, err)
	}
}

func TestDecoderPanicAndWarnings(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 25; i++ {
		_ = os.WriteFile(filepath.Join(root, fmt.Sprintf("%d.png", i)), nil, 0600)
	}
	var warnings atomic.Int32
	stats, err := scanTree(context.Background(), []string{root}, dimensionQuery{axis: "width", over: true, value: 1}, scanOptions{workers: 4,
		decode: func(string) (int, int, bool, error) { panic("bad codec") }, warning: func(error) { warnings.Add(1) }}, func(imageData) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	if stats.errors != 25 || warnings.Load() != 10 {
		t.Fatalf("errors %+v / warnings %d", stats, warnings.Load())
	}
}

func TestStrictAndSinkErrorsTerminate(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 100; i++ {
		_ = os.WriteFile(filepath.Join(root, fmt.Sprintf("%d.png", i)), nil, 0600)
	}
	want := errors.New("broken pipe")
	_, err := scanTree(context.Background(), []string{root}, dimensionQuery{axis: "width", over: true, value: 1}, scanOptions{workers: 4, decode: func(string) (int, int, bool, error) { return 2, 2, true, nil }}, func(imageData) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("expected sink error, got %v", err)
	}
	_, err = scanTree(context.Background(), []string{root}, dimensionQuery{axis: "width", over: true, value: 1}, scanOptions{workers: 4, strict: true, decode: func(string) (int, int, bool, error) { return 0, 0, false, fmt.Errorf("invalid") }}, func(imageData) error { return nil })
	if err == nil {
		t.Fatal("strict decoder error must fail")
	}
}

func TestCancellationAndMissingRoot(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := scanTree(ctx, []string{root}, dimensionQuery{}, scanOptions{workers: 2, decode: func(string) (int, int, bool, error) { return 0, 0, false, nil }}, func(imageData) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	_, err = scanTree(context.Background(), []string{root + "-missing"}, dimensionQuery{}, scanOptions{workers: 2, decode: func(string) (int, int, bool, error) { return 0, 0, false, nil }}, func(imageData) error { return nil })
	if err == nil {
		t.Fatal("expected error for missing input")
	}
}

func TestComparison(t *testing.T) {
	for _, tc := range []struct {
		q    dimensionQuery
		want bool
	}{
		{dimensionQuery{axis: "width", over: true, value: 10}, false},
		{dimensionQuery{axis: "width", over: true, equal: true, value: 10}, true},
		{dimensionQuery{axis: "height", over: false, value: 10}, true},
		{dimensionQuery{axis: "height", over: false, value: 5}, false},
	} {
		if got := tc.q.matches(10, 6); got != tc.want {
			t.Errorf("%+v -> %v", tc.q, got)
		}
	}
}
