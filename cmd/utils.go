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
	fs "io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/pkg/errors"
)

type imageData struct {
	name   string
	width  int
	height int
}

func parseSortBy() (sortBy, error) {
	InvalidSortBy := errors.New("invalid sort order provided. valid options: name, height, width")

	switch {
	case SortBy == "name":
		return name, nil
	case SortBy == "height":
		return height, nil
	case SortBy == "width":
		return width, nil
	default:
		return 0, InvalidSortBy
	}
}

func parseSortOrder() (sortOrder, error) {
	InvalidSortOrder := errors.New("invalid sort order provided. valid options: ascending, descending")

	switch {
	case SortOrder == "ascending":
		return ascending, nil
	case SortOrder == "descending":
		return descending, nil
	default:
		return -1, InvalidSortOrder
	}
}

func sortOutput(outputs []imageData) error {
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

func generateOutput(comparisonOperator compareType, compareValue int, fullPath string, height int, width int) imageData {
	switch {
	case OrEqual && comparisonOperator == widerthan && width >= compareValue,
		OrEqual && comparisonOperator == narrowerthan && width <= compareValue,
		OrEqual && comparisonOperator == tallerthan && height >= compareValue,
		OrEqual && comparisonOperator == shorterthan && height <= compareValue:
		return imageData{name: fullPath, width: width, height: height}
	case comparisonOperator == widerthan && width > compareValue,
		comparisonOperator == narrowerthan && width < compareValue,
		comparisonOperator == tallerthan && height > compareValue,
		comparisonOperator == shorterthan && height < compareValue:
		return imageData{name: fullPath, width: width, height: height}
	default:
		return imageData{}
	}
}

func scanFile(file fs.DirEntry, fileScans chan int, outputChannel chan<- imageData, scanDirectoryWaitGroup *sync.WaitGroup, comparisonOperator compareType, compareValue int, directory string) error {
	defer func() {
		<-fileScans
		scanDirectoryWaitGroup.Done()
	}()

	fileScans <- 1

	fileName := file.Name()
	fullPath := filepath.Join(directory, fileName)

	reader, err := os.Open(fullPath)
	if err != nil {
		return err
	}

	defer func() error {
		err := reader.Close()
		if err != nil {
			return err
		}

		return nil
	}()

	myImage, _, err := image.DecodeConfig(reader)
	if errors.Is(err, image.ErrFormat) {
		return nil
	} else if err != nil {
		return err
	}

	width := myImage.Width
	height := myImage.Height

	output := generateOutput(comparisonOperator, compareValue, fullPath, width, height)
	if (imageData{} == output) {
		err := errors.New("passed empty imageData{} to scanFile()")
		return err
	}

	if output.name != "" {
		outputChannel <- output
	}

	return nil
}

func scanDirectory(directoryScans chan int, fileScans chan int, outputChannel chan imageData, scanDirectoriesWaitGroup *sync.WaitGroup, comparisonOperator compareType, compareValue int, directory string) error {
	defer func() {
		<-directoryScans
		scanDirectoriesWaitGroup.Done()
	}()

	directoryScans <- 1

	var scanDirectoryWaitGroup sync.WaitGroup

	files, err := os.ReadDir(directory)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	for _, file := range files {
		scanDirectoryWaitGroup.Add(1)

		go scanFile(file, fileScans, outputChannel, &scanDirectoryWaitGroup, comparisonOperator, compareValue, directory)
	}

	scanDirectoryWaitGroup.Wait()

	return nil
}

func scanDirectories(directoryScans chan int, fileScans chan int, outputChannel chan imageData, imageSizesWaitGroup *sync.WaitGroup, comparisonOperator compareType, compareValue int, arguments []string, dir int) error {
	defer imageSizesWaitGroup.Done()

	var scanDirectoriesWaitGroup sync.WaitGroup

	if Recursive {
		directory := arguments[dir]

		filesystem := os.DirFS(directory)

		fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				scanDirectoriesWaitGroup.Add(1)
				fullPath := filepath.Join(directory, path)
				go scanDirectory(directoryScans, fileScans, outputChannel, &scanDirectoriesWaitGroup, comparisonOperator, compareValue, fullPath)
			}

			return nil
		})
	} else {
		scanDirectoriesWaitGroup.Add(1)
		err := scanDirectory(directoryScans, fileScans, outputChannel, &scanDirectoriesWaitGroup, comparisonOperator, compareValue, arguments[dir])
		if err != nil {
			return err
		}
	}

	scanDirectoriesWaitGroup.Wait()

	return nil
}

func ImageSizes(comparisonOperator compareType, arguments []string) error {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		return err
	}

	outputChannel := make(chan imageData)

	var imageSizesWaitGroup sync.WaitGroup

	directoryScans := make(chan int, maxDirectoryScans)
	fileScans := make(chan int, maxFileScans)

	for dir := 1; dir < len(arguments); dir++ {
		imageSizesWaitGroup.Add(1)
		go scanDirectories(directoryScans, fileScans, outputChannel, &imageSizesWaitGroup, comparisonOperator, compareValue, arguments, dir)
	}

	go func() {
		imageSizesWaitGroup.Wait()
		close(outputChannel)
		close(directoryScans)
		close(fileScans)
	}()

	var outputs []imageData

	for r := range outputChannel {
		outputs = append(outputs, r)
	}

	if !Unsorted {
		sortOutput(outputs)
	}

	if !Quiet {
		for o := 0; o < len(outputs); o++ {
			i := outputs[o]
			fmt.Printf("%v (%vx%v)\n", i.name, i.width, i.height)
		}
	}

	if Count {
		fmt.Printf("\n%v file(s) matched.\n", len(outputs))
	}

	fmt.Printf("")

	return nil
}
