/*
Copyright © 2025 Seednode <seednode@seedno.de>
*/

package main

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
	"time"
)

type sortDirection int

const (
	ascending sortDirection = iota
	descending
)

type sortKey int

const (
	name sortKey = iota
	height
	width
)

type compareType int

const (
	wider compareType = iota
	narrower
	taller
	shorter
)

type comparison struct {
	operator compareType
	value    int
}

type imageData struct {
	name   string
	width  int
	height int
}

func parseSortBy() sortKey {
	switch {
	case key == "name":
		return name
	case key == "height":
		return height
	case key == "width":
		return width
	default:
		fmt.Println(`Unknown key provided. Defaulting to "name".`)

		return name
	}
}

func parseSortOrder() sortDirection {
	switch {
	case order == "ascending" || order == "asc":
		return ascending
	case order == "descending" || order == "desc":
		return descending
	default:
		fmt.Println(`Unknown order provided. Defaulting to "ascending".`)

		return ascending
	}
}

func sortOutput(outputs []imageData) {
	sortBy, sortOrder := parseSortBy(), parseSortOrder()

	switch {
	case sortOrder == ascending && sortBy == height:
		sort.SliceStable(outputs, func(p, q int) bool {
			return outputs[p].height < outputs[q].height
		})
	case sortOrder == ascending && sortBy == width:
		sort.SliceStable(outputs, func(p, q int) bool {
			return outputs[p].width < outputs[q].width
		})
	case sortOrder == ascending && sortBy == name:
		sort.SliceStable(outputs, func(p, q int) bool {
			return outputs[p].name < outputs[q].name
		})
	case sortOrder == descending && sortBy == height:
		sort.SliceStable(outputs, func(p, q int) bool {
			return outputs[p].height > outputs[q].height
		})
	case sortOrder == descending && sortBy == width:
		sort.SliceStable(outputs, func(p, q int) bool {
			return outputs[p].width > outputs[q].width
		})
	case sortOrder == descending && sortBy == name:
		sort.SliceStable(outputs, func(p, q int) bool {
			return outputs[p].name > outputs[q].name
		})
	}
}

func walkPath(path string, compare *comparison, scans chan int, results chan<- imageData) error {
	scans <- 1

	defer func() {
		<-scans
	}()

	errs := make(chan error)
	done := make(chan bool, 1)

	nodes, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)

		go func(node fs.DirEntry) {
			defer wg.Done()

			fullPath := filepath.Join(path, node.Name())

			switch {
			case node.IsDir() && recursive:
				err := walkPath(fullPath, compare, scans, results)
				if err != nil {
					errs <- err

					return
				}
			case !node.IsDir():
				scans <- 1

				defer func() {
					<-scans
				}()

				file, err := os.Open(fullPath)
				if err != nil {
					errs <- err

					return
				}
				defer func() {
					err := file.Close()
					if err != nil {
						errs <- err

						return
					}
				}()

				decoded, _, err := image.DecodeConfig(file)
				if errors.Is(err, image.ErrFormat) {
					return
				} else if err != nil {
					errs <- err

					return
				}

				switch {
				case orEqual && compare.operator == wider && decoded.Width >= compare.value,
					orEqual && compare.operator == narrower && decoded.Width <= compare.value,
					orEqual && compare.operator == taller && decoded.Height >= compare.value,
					orEqual && compare.operator == shorter && decoded.Height <= compare.value,
					compare.operator == wider && decoded.Width > compare.value,
					compare.operator == narrower && decoded.Width < compare.value,
					compare.operator == taller && decoded.Height > compare.value,
					compare.operator == shorter && decoded.Height < compare.value:
					results <- imageData{name: fullPath, width: decoded.Width, height: decoded.Height}
				}
			}
		}(node)
	}

	go func() {
		wg.Wait()

		close(done)
	}()

Poll:
	for {
		select {
		case err := <-errs:
			return err
		case <-done:
			break Poll
		}
	}

	return nil
}

func imageSizes(compareOperator compareType, arguments []string) error {
	log.SetFlags(0)

	startTime := time.Now()

	if len(arguments) == 1 {
		arguments = append(arguments, ".")

		fmt.Println("No path specified. Defaulting to current directory.")
	}

	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		return err
	}

	compare := &comparison{
		operator: compareOperator,
		value:    compareValue,
	}

	results := make(chan imageData)
	errs := make(chan error)
	scanDone := make(chan bool)
	resultsDone := make(chan bool)

	var outputs []imageData

	go func() {
		for {
			select {
			case result := <-results:
				outputs = append(outputs, result)
			case <-scanDone:
				close(resultsDone)

				return
			}
		}
	}()

	var wg sync.WaitGroup

	scans := make(chan int, concurrency)

	for i := 1; i < len(arguments); i++ {
		wg.Add(1)

		go func(path string) {
			defer wg.Done()

			err := walkPath(path, compare, scans, results)
			if err != nil {
				errs <- err
			}
		}(arguments[i])
	}

	go func() {
		wg.Wait()

		close(scanDone)
	}()

Poll:
	for {
		select {
		case err := <-errs:
			return err
		case <-resultsDone:
			break Poll
		}
	}

	sortOutput(outputs)

	if verbose {
		for _, output := range outputs {
			fmt.Printf("%v (%vx%v)\n", output.name, output.width, output.height)
		}

		if len(outputs) != 0 {
			fmt.Println("")
		}

		fmt.Printf("%d file(s) matched in %v.\n",
			len(outputs),
			time.Since(startTime),
		)
	} else {
		for _, output := range outputs {
			fmt.Printf("%v\n", output.name)
		}
	}

	return nil
}
