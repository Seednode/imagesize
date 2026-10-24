package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIFlagsAnywhere(t *testing.T) {
	for _, args := range [][]string{
		{"width", "over", "512", "/p", "-r", "-c", "7", "-k", "height", "-o", "desc"},
		{"-c=7", "-r", "width", "over", "512", "-k=height", "-o=desc", "/p"},
	} {
		got, err := parseCLI(args)
		if err != nil {
			t.Fatal(err)
		}
		if got.workers != 7 || !got.recursive || got.sortKey != "height" || got.sortOrder != "desc" || len(got.positionals) != 4 {
			t.Fatalf("bad flags: %+v", got)
		}
	}
}
func TestCLIValidationAndHelp(t *testing.T) {
	for _, args := range [][]string{
		{"width", "sideways", "100"}, {"width", "over", "bad"}, {"height", "under", "100", "-c", "0"}, {"width", "under", "42", "--sort-key", "bad"},
	} {
		if err := run(context.Background(), args, &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
	var out bytes.Buffer
	if err := run(context.Background(), []string{"--help"}, &out, &bytes.Buffer{}); err != nil || !strings.Contains(out.String(), "sort-buffer") {
		t.Fatal(err, out.String())
	}
}
func TestCLIIntegration(t *testing.T) {
	root := t.TempDir()
	// This test intentionally uses a file with a .png extension and can run
	// against the real image decoder by constructing a valid PNG.
	path := filepath.Join(root, "valid.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	err = run(context.Background(), []string{"width", "over", "0", root, "-c", "2", "-k", "name"}, &out, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "valid.png") {
		t.Fatalf("unexpected stdout=%q stderr=%q", out.String(), stderr.String())
	}
}
