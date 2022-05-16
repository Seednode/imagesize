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

func ParseSortBy() (SortKey, error) {
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

func ParseSortOrder() (SortDirection, error) {
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

func SortOutput(outputs []imageData) error {
	sortBy, err := ParseSortBy()
	if err != nil {
		return err
	}

	sortOrder, err := ParseSortOrder()
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

func GenerateOutput(comparisonOperator CompareType, compareValue int, fullPath string, height int, width int) imageData {
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

func DecodeImage(fullPath string, reader io.Reader, outputChannel chan<- imageData, comparisonOperator CompareType, compareValue int) error {
	myImage, _, err := image.DecodeConfig(reader)
	if err != nil {
		return err
	}

	width := myImage.Width
	height := myImage.Height

	output := GenerateOutput(comparisonOperator, compareValue, fullPath, width, height)
	if (imageData{} == output) {
		err := errors.New("passed empty imageData{} to ScanFile()")
		return err
	}

	if output.name != "" {
		outputChannel <- output
	}

	return nil
}

func ReadFile(fullPath string) (*os.File, io.Reader, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, err
	}

	reader := file

	return file, reader, err
}

func ScanFile(fullPath string, comparisonOperator CompareType, compareValue int, fileScans chan int, outputChannel chan<- imageData, scanDirectoryWaitGroup *sync.WaitGroup) error {
	defer func() {
		<-fileScans
		scanDirectoryWaitGroup.Done()
	}()

	fileScans <- 1

	filePtr, reader, err := ReadFile(fullPath)
	if err != nil {
		return err
	}
	defer func() error {
		err := filePtr.Close()
		if err != nil {
			return err
		}

		return nil
	}()

	DecodeImage(fullPath, reader, outputChannel, comparisonOperator, compareValue)

	return nil
}

func ScanDirectory(directory string, comparisonOperator CompareType, compareValue int, directoryScans chan int, fileScans chan int, outputChannel chan imageData, scanDirectoriesWaitGroup *sync.WaitGroup) error {
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

		fullPath := filepath.Join(directory, file.Name())
		go ScanFile(fullPath, comparisonOperator, compareValue, fileScans, outputChannel, &scanDirectoryWaitGroup)
	}

	scanDirectoryWaitGroup.Wait()

	return nil
}

func ScanDirectories(directory string, comparisonOperator CompareType, compareValue int, directoryScans chan int, fileScans chan int, outputChannel chan imageData, imageSizesWaitGroup *sync.WaitGroup) error {
	defer imageSizesWaitGroup.Done()

	var ScanDirectoriesWaitGroup sync.WaitGroup

	if Recursive {
		filesystem := os.DirFS(directory)

		fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				ScanDirectoriesWaitGroup.Add(1)

				fullPath := filepath.Join(directory, path)
				go ScanDirectory(fullPath, comparisonOperator, compareValue, directoryScans, fileScans, outputChannel, &ScanDirectoriesWaitGroup)
			}

			return nil
		})
	} else {
		ScanDirectoriesWaitGroup.Add(1)
		err := ScanDirectory(directory, comparisonOperator, compareValue, directoryScans, fileScans, outputChannel, &ScanDirectoriesWaitGroup)
		if err != nil {
			return err
		}
	}

	ScanDirectoriesWaitGroup.Wait()

	return nil
}

func ImageSizes(comparisonOperator CompareType, arguments []string) error {
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
		directory := arguments[dir]
		go ScanDirectories(directory, comparisonOperator, compareValue, directoryScans, fileScans, outputChannel, &imageSizesWaitGroup)
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
		SortOutput(outputs)
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
