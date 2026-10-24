// Copyright 2026 Seednode. Licensed under the MIT license.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const releaseVersion = "1.3.1"

type cliOptions struct {
	workers                                          int
	recursive, equal, verbose, version, help, strict bool
	sortKey, sortOrder                               string
	sortBuffer                                       int
	positionals                                      []string
}

func defaults() cliOptions {
	workers := runtime.NumCPU() * 2
	if workers > 32 {
		workers = 32
	}
	if workers < 2 {
		workers = 2
	}
	return cliOptions{workers: workers, sortKey: "name", sortOrder: "ascending", sortBuffer: 50000}
}

const usage = `imagesize - list images matching dimension constraints

Usage:
  imagesize width  over  <pixels> [paths...] [flags]
  imagesize width  under <pixels> [paths...] [flags]
  imagesize height over  <pixels> [paths...] [flags]
  imagesize height under <pixels> [paths...] [flags]

When no path is given, the current directory is used.

Flags (may appear anywhere):
  -r, --recursive              include subdirectories
  -e, --or-equal               include images equal to the threshold
  -c, --max-concurrency N      decode workers (default up to 32; limit 1..256)
  -k, --sort-key KEY           name, width, height, none (default name)
  -o, --sort-order DIRECTION   asc/ascending, desc/descending
      --sort-buffer N          max matches held in RAM per sort run (default 50000)
      --strict                 fail on file/directory errors (default: warn & continue)
  -v, --verbose                show dimensions and number of matches
  -V, --version                show version
  -h, --help                   show this message

Supported formats: JPEG, PNG, GIF, BMP, WebP, JXL, AVIF, HEIF/HEIC.
Sorting spills to temporary files for very large result sets. --sort-key none
prints matches as they arrive, avoiding the sort files and result buffering.
`

// parseCLI supports flags before or after commands and paths, preserving
// Cobra's practical interspersed-flag behavior without a third-party CLI layer.
func parseCLI(args []string) (cliOptions, error) {
	o := defaults()
	flagsEnabled := true
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" && flagsEnabled {
			flagsEnabled = false
			continue
		}
		if !flagsEnabled || !strings.HasPrefix(a, "-") || a == "-" {
			o.positionals = append(o.positionals, a)
			continue
		}
		name, val, hasValue := strings.Cut(a, "=")
		needValue := func() (string, error) {
			if hasValue {
				return val, nil
			}
			i++
			if i >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			return args[i], nil
		}
		switch name {
		case "--recursive", "-r":
			o.recursive = true
		case "--or-equal", "-e":
			o.equal = true
		case "--verbose", "-v":
			o.verbose = true
		case "--strict":
			o.strict = true
		case "--version", "-V":
			o.version = true
		case "--help", "-h":
			o.help = true
		case "--max-concurrency", "-c":
			s, err := needValue()
			if err != nil {
				return o, err
			}
			o.workers, err = strconv.Atoi(s)
			if err != nil {
				return o, fmt.Errorf("invalid concurrency %q: %w", s, err)
			}
		case "--sort-key", "-k":
			s, err := needValue()
			if err != nil {
				return o, err
			}
			o.sortKey = strings.ToLower(s)
		case "--sort-order", "-o":
			s, err := needValue()
			if err != nil {
				return o, err
			}
			o.sortOrder = strings.ToLower(s)
		case "--sort-buffer":
			s, err := needValue()
			if err != nil {
				return o, err
			}
			o.sortBuffer, err = strconv.Atoi(s)
			if err != nil {
				return o, err
			}
		default:
			// Permit compact combinations of boolean aliases, such as -rev.
			if len(name) > 2 && strings.HasPrefix(name, "-") && !strings.HasPrefix(name, "--") && !hasValue {
				for _, ch := range name[1:] {
					switch ch {
					case 'r':
						o.recursive = true
					case 'e':
						o.equal = true
					case 'v':
						o.verbose = true
					case 'V':
						o.version = true
					case 'h':
						o.help = true
					default:
						return o, fmt.Errorf("unknown flag -%c", ch)
					}
				}
			} else {
				return o, fmt.Errorf("unknown flag %q", a)
			}
		}
	}
	return o, nil
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	opts, err := parseCLI(args)
	if err != nil {
		return err
	}
	if opts.version {
		_, _ = fmt.Fprintf(stdout, "imagesize v%s\n", releaseVersion)
		return nil
	}
	if opts.help {
		_, _ = fmt.Fprint(stdout, usage)
		return nil
	}
	if len(opts.positionals) < 3 {
		return fmt.Errorf("missing dimension or comparison (see --help)")
	}
	axis, direction, threshold := opts.positionals[0], opts.positionals[1], opts.positionals[2]
	if axis != "width" && axis != "height" {
		return fmt.Errorf("unknown dimension %q: expected width or height", axis)
	}
	if direction != "under" && direction != "over" {
		return fmt.Errorf("unknown comparison %q: expected under or over", direction)
	}
	limit, err := strconv.Atoi(threshold)
	if err != nil || limit < 0 {
		return fmt.Errorf("invalid pixel threshold %q", threshold)
	}
	if opts.workers < 1 || opts.workers > 256 {
		return fmt.Errorf("max concurrency must be between 1 and 256")
	}
	if opts.sortBuffer < 1 {
		return fmt.Errorf("sort buffer must be >= 1")
	}
	switch opts.sortKey {
	case "name", "width", "height", "none":
	default:
		return fmt.Errorf("unknown sort key %q", opts.sortKey)
	}
	var descending bool
	switch opts.sortOrder {
	case "asc", "ascending":
	case "desc", "descending":
		descending = true
	default:
		return fmt.Errorf("unknown sort order %q", opts.sortOrder)
	}
	roots := opts.positionals[3:]
	if len(roots) == 0 {
		roots = []string{"."}
	}

	collector, err := newCollector(stdout, opts.sortKey, descending, opts.verbose, opts.sortBuffer)
	if err != nil {
		return err
	}
	defer collector.close()
	start := time.Now()
	stats, err := scanTree(ctx, roots, dimensionQuery{axis: axis, over: direction == "over", equal: opts.equal, value: limit}, scanOptions{
		recursive: opts.recursive, strict: opts.strict, workers: opts.workers, decode: imageDimensions,
		warning: func(err error) { fmt.Fprintf(stderr, "warning: %v\n", err) },
	}, collector.add)
	if err != nil {
		return err
	}
	if err := collector.finish(); err != nil {
		return err
	}
	if opts.verbose {
		if stats.matched != 0 {
			fmt.Fprintln(stdout)
		}
		fmt.Fprintf(stdout, "%d file(s) matched in %v.\n", stats.matched, time.Since(start))
		fmt.Fprintf(stderr, "Scanned %d file(s), recognized %d image(s); encountered %d error(s).\n", stats.files, stats.images, stats.errors)
	}
	if stats.errors > 10 {
		fmt.Fprintf(stderr, "warning: %d additional errors omitted\n", stats.errors-10)
	}
	return nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "interrupted")
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "imagesize:", err)
		os.Exit(1)
	}
}
