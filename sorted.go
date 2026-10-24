// Copyright 2026 Seednode. Licensed under the MIT license.
package main

import (
	"bufio"
	"container/heap"
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type collector struct {
	writer  *bufio.Writer
	key     string
	desc    bool
	verbose bool
	limit   int
	buffer  []imageData
	dir     string
	runs    []string
	count   int64
}

func newCollector(out io.Writer, key string, desc, verbose bool, limit int) (*collector, error) {
	if limit < 1 {
		return nil, fmt.Errorf("sort buffer must be >= 1")
	}
	return &collector{writer: bufio.NewWriterSize(out, 128*1024), key: key, desc: desc, verbose: verbose, limit: limit}, nil
}

func (c *collector) less(a, b imageData) bool {
	cmp := 0
	switch c.key {
	case "height":
		cmp = compareInt(a.Height, b.Height)
	case "width":
		cmp = compareInt(a.Width, b.Width)
	default:
		cmp = compareString(a.Name, b.Name)
	}
	if c.desc {
		cmp = -cmp
	}
	if cmp == 0 {
		cmp = compareString(a.Name, b.Name)
	} // deterministic ties
	if cmp == 0 {
		cmp = compareInt(a.Width, b.Width)
	}
	if cmp == 0 {
		cmp = compareInt(a.Height, b.Height)
	}
	return cmp < 0
}

func compareInt(a, b int) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
func compareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func (c *collector) add(item imageData) error {
	c.count++
	if c.key == "none" {
		return c.print(item)
	}
	c.buffer = append(c.buffer, item)
	if len(c.buffer) >= c.limit {
		return c.spill()
	}
	return nil
}

func (c *collector) print(item imageData) error {
	if c.verbose {
		_, err := fmt.Fprintf(c.writer, "%s (%dx%d)\n", item.Name, item.Width, item.Height)
		return err
	}
	_, err := fmt.Fprintln(c.writer, item.Name)
	return err
}

func (c *collector) spill() (err error) {
	if len(c.buffer) == 0 {
		return nil
	}
	sort.Slice(c.buffer, func(i, j int) bool { return c.less(c.buffer[i], c.buffer[j]) })
	if c.dir == "" {
		c.dir, err = os.MkdirTemp("", "imagesize-sort-*")
		if err != nil {
			return err
		}
	}
	f, err := os.CreateTemp(c.dir, "run-*.gob")
	if err != nil {
		return err
	}
	defer f.Close()
	// Register before writing, ensuring cleanup on any error.
	c.runs = append(c.runs, f.Name())
	buf := bufio.NewWriterSize(f, 128*1024)
	enc := gob.NewEncoder(buf)
	for _, item := range c.buffer {
		if err := enc.Encode(item); err != nil {
			return err
		}
	}
	if err := buf.Flush(); err != nil {
		return err
	}
	c.buffer = c.buffer[:0]
	return nil
}

type runReader struct {
	file    *os.File
	decoder *gob.Decoder
}

type heapItem struct {
	data imageData
	from int
}

type mergeHeap struct {
	items []heapItem
	less  func(a, b imageData) bool
}

func (h mergeHeap) Len() int           { return len(h.items) }
func (h mergeHeap) Less(i, j int) bool { return h.less(h.items[i].data, h.items[j].data) }
func (h mergeHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *mergeHeap) Push(x any)        { h.items = append(h.items, x.(heapItem)) }
func (h *mergeHeap) Pop() any {
	n := len(h.items)
	x := h.items[n-1]
	h.items = h.items[:n-1]
	return x
}

func (c *collector) finish() error {
	if c.key == "none" {
		return c.writer.Flush()
	}
	if len(c.runs) == 0 {
		sort.Slice(c.buffer, func(i, j int) bool { return c.less(c.buffer[i], c.buffer[j]) })
		for _, item := range c.buffer {
			if err := c.print(item); err != nil {
				return err
			}
		}
		return c.writer.Flush()
	}
	if err := c.spill(); err != nil {
		return err
	}
	readers := make([]runReader, len(c.runs))
	defer func() {
		for _, r := range readers {
			if r.file != nil {
				_ = r.file.Close()
			}
		}
	}()
	h := &mergeHeap{less: c.less}
	for i, path := range c.runs {
		f, err := os.Open(filepath.Clean(path))
		if err != nil {
			return err
		}
		readers[i] = runReader{file: f, decoder: gob.NewDecoder(bufio.NewReaderSize(f, 128*1024))}
		var item imageData
		if err := readers[i].decoder.Decode(&item); err != nil {
			if err == io.EOF {
				continue
			}
			return err
		}
		heap.Push(h, heapItem{data: item, from: i})
	}
	for h.Len() > 0 {
		item := heap.Pop(h).(heapItem)
		if err := c.print(item.data); err != nil {
			return err
		}
		var next imageData
		err := readers[item.from].decoder.Decode(&next)
		if err == io.EOF {
			continue
		}
		if err != nil {
			return err
		}
		heap.Push(h, heapItem{data: next, from: item.from})
	}
	return c.writer.Flush()
}

func (c *collector) close() error {
	if c.dir == "" {
		return nil
	}
	return os.RemoveAll(c.dir)
}
