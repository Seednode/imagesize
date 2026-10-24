// Copyright 2026 Seednode. Licensed under the MIT license.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type imageData struct {
	Name          string
	Width, Height int
}

type dimensionQuery struct {
	axis        string
	over, equal bool
	value       int
}

func (q dimensionQuery) matches(w, h int) bool {
	v := w
	if q.axis == "height" {
		v = h
	}
	if q.over {
		return v > q.value || q.equal && v == q.value
	}
	return v < q.value || q.equal && v == q.value
}

type scanOptions struct {
	recursive bool
	strict    bool
	workers   int
	// Decoder is injectable to allow deterministic load and cancellation tests.
	decode func(string) (int, int, bool, error)
	// Warnings are serialized, capped, and sent to stderr by the caller.
	warning func(error)
}

type scanStats struct {
	files   int64
	images  int64
	matched int64
	errors  int64
}

// scanTree has O(workers) goroutines and a bounded queue regardless of file count.
// One directory producer is the sole owner of traversal state. Workers never
// wait on a parent worker's semaphore, and cancellation wakes every blocked send.
func scanTree(ctx context.Context, roots []string, q dimensionQuery, opts scanOptions, emit func(imageData) error) (scanStats, error) {
	var stats scanStats
	if opts.workers < 1 || opts.workers > 256 {
		return stats, fmt.Errorf("max concurrency must be between 1 and 256")
	}
	if opts.decode == nil || emit == nil {
		return stats, fmt.Errorf("scan decoder and result handler are required")
	}
	if len(roots) == 0 {
		roots = []string{"."}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan string, opts.workers*2)
	producerErr := make(chan error, 1)
	var warnMu sync.Mutex
	var warnings int64
	warn := func(err error) {
		if err == nil {
			return
		}
		atomic.AddInt64(&stats.errors, 1)
		warnMu.Lock()
		defer warnMu.Unlock()
		warnings++
		if opts.warning != nil && warnings <= 10 {
			opts.warning(err)
		}
	}
	var outputMu sync.Mutex
	var firstOutputErr error
	setOutputErr := func(err error) {
		outputMu.Lock()
		if firstOutputErr == nil {
			firstOutputErr = err
			cancel()
		}
		outputMu.Unlock()
	}

	var wg sync.WaitGroup
	wg.Add(opts.workers)
	for i := 0; i < opts.workers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case path, ok := <-jobs:
					if !ok {
						return
					}
					atomic.AddInt64(&stats.files, 1)
					w, h, supported, err := safeDimensions(opts.decode, path)
					if err != nil {
						warn(fmt.Errorf("%s: %w", path, err))
						if opts.strict {
							setOutputErr(fmt.Errorf("%s: %w", path, err))
						}
						continue
					}
					if !supported {
						continue
					}
					atomic.AddInt64(&stats.images, 1)
					if !q.matches(w, h) {
						continue
					}
					outputMu.Lock()
					if firstOutputErr == nil && ctx.Err() == nil {
						if err := emit(imageData{Name: path, Width: w, Height: h}); err != nil {
							firstOutputErr = err
							cancel()
						} else {
							atomic.AddInt64(&stats.matched, 1)
						}
					}
					outputMu.Unlock()
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		producerErr <- walkRoots(ctx, roots, opts.recursive, opts.strict, func(err error) { warn(err) }, func(path string) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case jobs <- path:
				return nil
			}
		})
	}()

	wg.Wait()
	walkErr := <-producerErr
	outputMu.Lock()
	outputErr := firstOutputErr
	outputMu.Unlock()
	if outputErr != nil {
		return stats, outputErr
	}
	if walkErr != nil && !errors.Is(walkErr, context.Canceled) {
		return stats, walkErr
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	return stats, nil
}

func safeDimensions(decode func(string) (int, int, bool, error), path string) (w, h int, supported bool, err error) {
	defer func() {
		if v := recover(); v != nil {
			w, h, supported = 0, 0, false
			err = fmt.Errorf("image decoder panic: %v", v)
		}
	}()
	return decode(path)
}

func walkRoots(ctx context.Context, roots []string, recursive, strict bool, warning func(error), submit func(string) error) error {
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			return err
		}
		info, err := os.Stat(root)
		if err != nil {
			return fmt.Errorf("input %q: %w", root, err)
		}
		if info.Mode().IsRegular() {
			if err := submit(root); err != nil {
				return err
			}
			continue
		}
		if !info.IsDir() {
			return fmt.Errorf("input %q is neither a regular file nor a directory", root)
		}
		if err := walkDirectory(ctx, root, recursive, strict, warning, submit); err != nil {
			return err
		}
	}
	return nil
}

// ReadDir(n) caps each directory batch. We do not call os.ReadDir(path), which
// allocates memory proportional to the largest directory. Symlinked directories
// are deliberately not followed, preventing cycles; file symlinks are supported.
func walkDirectory(ctx context.Context, path string, recursive, strict bool, warning func(error), submit func(string) error) error {
	file, err := os.Open(path)
	if err != nil {
		if strict {
			return fmt.Errorf("open directory %q: %w", path, err)
		}
		warning(fmt.Errorf("open directory %q: %w", path, err))
		return nil
	}
	defer file.Close()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := file.ReadDir(256)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			full := filepath.Join(path, entry.Name())
			if entry.IsDir() {
				if recursive {
					if err := walkDirectory(ctx, full, recursive, strict, warning, submit); err != nil {
						return err
					}
				}
				continue
			}
			// Do not attempt to open sockets, FIFOs, block devices, or symlinked directories.
			if entry.Type().IsRegular() {
				if err := submit(full); err != nil {
					return err
				}
			} else if entry.Type()&fs.ModeSymlink != 0 {
				if target, err := os.Stat(full); err == nil && target.Mode().IsRegular() {
					if err := submit(full); err != nil {
						return err
					}
				}
			} else if entry.Type() == 0 {
				// DirEntry.Type may be unknown on some filesystems.
				if info, err := entry.Info(); err == nil && info.Mode().IsRegular() {
					if err := submit(full); err != nil {
						return err
					}
				}
			}
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			if strict {
				return fmt.Errorf("read directory %q: %w", path, readErr)
			}
			warning(fmt.Errorf("read directory %q: %w", path, readErr))
			return nil
		}
	}
}
