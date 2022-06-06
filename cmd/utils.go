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

type ImageData struct {
	name   string
	width  int
	height int
}

func parseSortBy() (SortKey, error) {
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

func parseSortOrder() (SortDirection, error) {
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

func generateOutput(comparisonOperator CompareType, compareValue int, fullPath string, width int, height int) ImageData {
	switch {
	case OrEqual && comparisonOperator == widerthan && width >= compareValue,
		OrEqual && comparisonOperator == narrowerthan && width <= compareValue,
		OrEqual && comparisonOperator == tallerthan && height >= compareValue,
		OrEqual && comparisonOperator == shorterthan && height <= compareValue,
		comparisonOperator == narrowerthan && width < compareValue,
		comparisonOperator == tallerthan && height > compareValue,
		comparisonOperator == shorterthan && height < compareValue:
		return ImageData{name: fullPath, width: width, height: height}
	default:
		return ImageData{}
	}
}

func decodeImage(fullPath string, reader io.Reader, outputChannel chan<- ImageData, comparisonOperator CompareType, compareValue int) error {
	myImage, _, err := image.DecodeConfig(reader)
	if errors.Is(err, image.ErrFormat) {
		return nil
	} else if err != nil {
		return err
	}

	width := myImage.Width
	height := myImage.Height

	output := generateOutput(comparisonOperator, compareValue, fullPath, width, height)
	if (ImageData{} == output) {
		err := errors.New("passed empty ImageData{} to ScanFile()")
		return err
	}

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

func scanFile(fullPath string, comparisonOperator CompareType, compareValue int, fileScans chan int, outputChannel chan<- ImageData, scanDirectoryWaitGroup *sync.WaitGroup) error {
	defer func() {
		<-fileScans
		scanDirectoryWaitGroup.Done()
	}()

	fileScans <- 1

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

	err = decodeImage(fullPath, reader, outputChannel, comparisonOperator, compareValue)
	if err != nil {
		return err
	}

	return nil
}

func scanDirectory(directory string, comparisonOperator CompareType, compareValue int, directoryScans chan int, fileScans chan int, outputChannel chan ImageData, scanDirectoriesWaitGroup *sync.WaitGroup) error {
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
		go func() {
			err := scanFile(fullPath, comparisonOperator, compareValue, fileScans, outputChannel, &scanDirectoryWaitGroup)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

	scanDirectoryWaitGroup.Wait()

	return nil
}

func scanDirectories(directory string, comparisonOperator CompareType, compareValue int, directoryScans chan int, fileScans chan int, outputChannel chan ImageData, imageSizesWaitGroup *sync.WaitGroup) error {
	defer imageSizesWaitGroup.Done()

	var ScanDirectoriesWaitGroup sync.WaitGroup

	if Recursive {
		filesystem := os.DirFS(directory)

		err := fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				ScanDirectoriesWaitGroup.Add(1)

				fullPath := filepath.Join(directory, path)
				go func() {
					err := scanDirectory(fullPath, comparisonOperator, compareValue, directoryScans, fileScans, outputChannel, &ScanDirectoriesWaitGroup)
					if err != nil {
						fmt.Println(err)
					}
				}()
			}

			return nil
		})
		if err != nil {
			fmt.Println(err)
		}
	} else {
		ScanDirectoriesWaitGroup.Add(1)
		err := scanDirectory(directory, comparisonOperator, compareValue, directoryScans, fileScans, outputChannel, &ScanDirectoriesWaitGroup)
		if err != nil {
			return err
		}
	}

	ScanDirectoriesWaitGroup.Wait()

	return nil
}

func ImageSizes(comparisonOperator CompareType, arguments []string) {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		fmt.Println(err)
	}

	outputChannel := make(chan ImageData)

	var imageSizesWaitGroup sync.WaitGroup

	directoryScans := make(chan int, maxDirectoryScans)
	fileScans := make(chan int, maxFileScans)

	for dir := 1; dir < len(arguments); dir++ {
		imageSizesWaitGroup.Add(1)
		directory := arguments[dir]
		go func() {
			err := scanDirectories(directory, comparisonOperator, compareValue, directoryScans, fileScans, outputChannel, &imageSizesWaitGroup)
			if err != nil {
				fmt.Println(err)
			}
		}()
	}

	go func() {
		imageSizesWaitGroup.Wait()
		close(outputChannel)
		close(directoryScans)
		close(fileScans)
	}()

	var outputs []ImageData

	for r := range outputChannel {
		outputs = append(outputs, r)
	}

	if !Unsorted {
		err := sortOutput(outputs)
		if err != nil {
			fmt.Println(err)
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
}
