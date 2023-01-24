/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

type sortDirection int

const (
	invalidSortDirection sortDirection = iota
	ascending
	descending
)

type sortKey int

const (
	invalidSortKey sortKey = iota
	name
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

type Comparison struct {
	operator compareType
	value    int
}

type ImageData struct {
	name   string
	width  int
	height int
}

type Scans struct {
	directories chan int
	files       chan int
}

func (s *Scans) Close() {
	close(s.directories)
	close(s.files)
}

func parseSortBy() (sortKey, error) {
	err := errors.New("invalid sort key provided. valid options: name, height, width")

	switch {
	case SortBy == "name":
		return name, nil
	case SortBy == "height":
		return height, nil
	case SortBy == "width":
		return width, nil
	default:
		return invalidSortKey, err
	}
}

func parseSortOrder() (sortDirection, error) {
	err := errors.New("invalid sort order provided. valid options: ascending, descending")

	switch {
	case SortOrder == "ascending":
		return ascending, nil
	case SortOrder == "descending":
		return descending, nil
	default:
		return invalidSortDirection, err
	}
}

func sortOutput(outputs []ImageData) error {
	sortBy, err := parseSortBy()
	if err != nil {
		return err
	}

	sortOrder, err := parseSortOrder()
	if err != nil {
		return err
	}

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

	return nil
}

func generateOutput(compare *Comparison, fullPath string, width int, height int) ImageData {
	switch {
	case OrEqual && compare.operator == wider && width >= compare.value,
		OrEqual && compare.operator == narrower && width <= compare.value,
		OrEqual && compare.operator == taller && height >= compare.value,
		OrEqual && compare.operator == shorter && height <= compare.value,
		compare.operator == wider && width > compare.value,
		compare.operator == narrower && width < compare.value,
		compare.operator == taller && height > compare.value,
		compare.operator == shorter && height < compare.value:
		return ImageData{name: fullPath, width: width, height: height}
	default:
		return ImageData{}
	}
}

func decodeImage(fullPath string, reader io.Reader, outputChannel chan<- ImageData, compare *Comparison) error {
	myImage, _, err := image.DecodeConfig(reader)
	if errors.Is(err, image.ErrFormat) {
		return nil
	} else if err != nil {
		return err
	}

	width := myImage.Width
	height := myImage.Height

	output := generateOutput(compare, fullPath, width, height)

	if output.name != "" {
		outputChannel <- output
	}

	return nil
}

func readFile(fullPath string) (*os.File, io.Reader, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, err
	}

	reader := file

	return file, reader, err
}

func scanFile(fullPath string, compare *Comparison, scans *Scans, outputChannel chan<- ImageData, wg *sync.WaitGroup) error {
	defer func() {
		<-scans.files
		wg.Done()
	}()

	scans.files <- 1

	filePtr, reader, err := readFile(fullPath)
	if err != nil {
		return err
	}
	defer func() {
		err := filePtr.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	err = decodeImage(fullPath, reader, outputChannel, compare)
	if err != nil {
		return err
	}

	return nil
}

func scanDirectory(directory string, compare *Comparison, scans *Scans, outputChannel chan ImageData, wg *sync.WaitGroup) error {
	defer func() {
		<-scans.directories
		wg.Done()
	}()

	scans.directories <- 1

	var wg2 sync.WaitGroup

	files, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		wg2.Add(1)

		fullPath := filepath.Join(directory, file.Name())
		go func() {
			err := scanFile(fullPath, compare, scans, outputChannel, &wg2)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

	wg2.Wait()

	return nil
}

func scanDirectories(directory string, compare *Comparison, scans *Scans, outputChannel chan ImageData, wg *sync.WaitGroup) error {
	defer wg.Done()

	var wg2 sync.WaitGroup

	if Recursive {
		filesystem := os.DirFS(directory)

		err := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				wg2.Add(1)

				fullPath := filepath.Join(directory, path)
				go func() {
					err := scanDirectory(fullPath, compare, scans, outputChannel, &wg2)
					if err != nil {
						fmt.Println(err)
					}
				}()
			}

			return nil
		})
		if err != nil {
			return err
		}
	} else {
		wg2.Add(1)
		err := scanDirectory(directory, compare, scans, outputChannel, &wg2)
		if err != nil {
			return err
		}
	}

	wg2.Wait()

	return nil
}

func ImageSizes(compareOperator compareType, arguments []string) error {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		return err
	}

	compare := &Comparison{
		operator: compareOperator,
		value:    compareValue,
	}

	outputChannel := make(chan ImageData)

	var wg sync.WaitGroup

	scans := &Scans{
		directories: make(chan int, maxDirectoryScans),
		files:       make(chan int, maxFileScans),
	}

	for i := 1; i < len(arguments); i++ {
		wg.Add(1)
		directory := arguments[i]
		go func() {
			err := scanDirectories(directory, compare, scans, outputChannel, &wg)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

	go func() {
		wg.Wait()
		close(outputChannel)
		scans.Close()
	}()

	var outputs []ImageData

	for r := range outputChannel {
		outputs = append(outputs, r)
	}

	if !Unsorted {
		err := sortOutput(outputs)
		if err != nil {
			return err
		}
	}

	switch {
	case !Quiet && Verbose:
		for o := 0; o < len(outputs); o++ {
			i := outputs[o]
			fmt.Printf("%v (%vx%v)\n", i.name, i.width, i.height)
		}
	case !Quiet && !Verbose:
		for o := 0; o < len(outputs); o++ {
			i := outputs[o]
			fmt.Printf("%v\n", i.name)
		}
	default:
	}

	if Count {
		fmt.Printf("\n%v file(s) matched.\n", len(outputs))
	}

	fmt.Printf("")

	return nil
}
