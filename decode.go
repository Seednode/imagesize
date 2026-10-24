// Copyright 2026 Seednode. Licensed under the MIT license.
package main

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/heic"
	"github.com/gen2brain/jpegxl"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/vp8l"
	_ "golang.org/x/image/webp"
)

// imageDimensions reads metadata, never full decoded pixel arrays. Proprietary
// container decoders are only attempted for plausible formats, and each attempt
// starts at offset zero (the old fallback reused an already-consumed reader).
func imageDimensions(path string) (int, int, bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, false, nil
		} // removed during scan
		return 0, 0, false, err
	}
	defer f.Close()

	var header [64]byte
	n, readErr := f.Read(header[:])
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return 0, 0, false, readErr
	}
	first := header[:n]
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return 0, 0, false, err
	}

	cfg, _, err := image.DecodeConfig(f)
	if err == nil {
		return checkedSize(cfg.Width, cfg.Height)
	}
	// For a file that clearly contains a supported image but has invalid metadata,
	// return the decode error. For ordinary unrelated files, silently skip.
	formatError := !errors.Is(err, image.ErrFormat)
	ext := strings.ToLower(filepath.Ext(path))
	isJXL := ext == ".jxl" || bytes.HasPrefix(first, []byte{0xff, 0x0a}) || bytes.HasPrefix(first, []byte{0, 0, 0, 12, 'J', 'X', 'L', ' ', 13, 10, 135, 10})
	isBMFF := len(first) >= 12 && bytes.Equal(first[4:8], []byte("ftyp"))
	isAVIF := ext == ".avif" || isBMFF && (hasBrand(first, "avif") || hasBrand(first, "avis"))
	isHEIC := ext == ".heic" || ext == ".heif" || isBMFF && (hasBrand(first, "heic") || hasBrand(first, "heix") || hasBrand(first, "hevc") || hasBrand(first, "hevx") || hasBrand(first, "mif1") || hasBrand(first, "msf1"))

	var lastErr error
	try := func(decode func(io.Reader) (image.Config, error)) (int, int, bool) {
		if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
			lastErr = seekErr
			return 0, 0, false
		}
		c, decodeErr := decode(f)
		if decodeErr != nil {
			lastErr = decodeErr
			return 0, 0, false
		}
		if c.Width < 1 || c.Height < 1 {
			lastErr = fmt.Errorf("invalid image dimensions %dx%d", c.Width, c.Height)
			return 0, 0, false
		}
		return c.Width, c.Height, true
	}
	if isJXL {
		if w, h, ok := try(jpegxl.DecodeConfig); ok {
			return w, h, true, nil
		}
	}
	if isAVIF {
		if w, h, ok := try(avif.DecodeConfig); ok {
			return w, h, true, nil
		}
	}
	if isHEIC {
		if w, h, ok := try(heic.DecodeConfig); ok {
			return w, h, true, nil
		}
	}
	if isJXL || isAVIF || isHEIC {
		return 0, 0, false, fmt.Errorf("decode image header: %w", lastErr)
	}
	if formatError && (isCommonImageExtension(ext) || hasStandardSignature(first)) {
		return 0, 0, false, err
	}
	return 0, 0, false, nil
}

func hasBrand(header []byte, brand string) bool {
	// The first few brands follow the ftyp box header. Only inspect complete
	// four-byte slots; never match arbitrary substrings in unrelated payload.
	if len(header) < 16 {
		return false
	}
	for i := 8; i+4 <= len(header); i += 4 {
		if string(header[i:i+4]) == brand {
			return true
		}
	}
	return false
}

func checkedSize(w, h int) (int, int, bool, error) {
	if w <= 0 || h <= 0 {
		return 0, 0, false, fmt.Errorf("invalid image dimensions %dx%d", w, h)
	}
	return w, h, true, nil
}

// A truncated unrelated file (including an empty file) is not an image error.
// Only report corrupt common image headers or files with image extensions.
func isCommonImageExtension(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".tif", ".tiff":
		return true
	}
	return false
}

func hasStandardSignature(data []byte) bool {
	return bytes.HasPrefix(data, []byte{0xff, 0xd8}) ||
		bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) ||
		bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a")) ||
		bytes.HasPrefix(data, []byte("BM")) ||
		(len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP")
}
