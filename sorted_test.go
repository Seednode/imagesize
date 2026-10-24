package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestCollectorSpillAndMerge(t *testing.T) {
	for _, key := range []string{"name", "width", "height"} {
		for _, descending := range []bool{false, true} {
			var b bytes.Buffer
			c, err := newCollector(&b, key, descending, true, 7)
			if err != nil {
				t.Fatal(err)
			}
			var data []imageData
			for i := 99; i >= 0; i-- {
				data = append(data, imageData{Name: fmt.Sprintf("img%03d.png", i), Width: i % 19, Height: i % 23})
			}
			for _, d := range data {
				if err := c.add(d); err != nil {
					t.Fatal(err)
				}
			}
			if len(c.runs) == 0 {
				t.Fatal("spill not triggered")
			}
			if err := c.finish(); err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(b.String()), "\n")
			if len(lines) != 100 {
				t.Fatalf("expected 100 lines, got %d", len(lines))
			}
			for i := 1; i < len(lines); i++ {
				// Validate output order via the same comparator but independent input lookup.
				find := func(line string) imageData {
					for _, d := range data {
						if strings.HasPrefix(line, d.Name+" (") {
							return d
						}
					}
					t.Fatalf("missing item: %s", line)
					return imageData{}
				}
				prev, curr := find(lines[i-1]), find(lines[i])
				if c.less(curr, prev) {
					t.Fatalf("unsorted %s %v: %v then %v", key, descending, prev, curr)
				}
			}
			if err := c.close(); err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestCollectorNoSort(t *testing.T) {
	var out bytes.Buffer
	c, _ := newCollector(&out, "none", false, false, 1)
	defer c.close()
	_ = c.add(imageData{Name: "z"})
	_ = c.add(imageData{Name: "a"})
	if err := c.finish(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "z\na\n" {
		t.Fatalf("got %q", out.String())
	}
}
