/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"errors"
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
)

type imageData struct {
	name   string
	width  int
	height int
}

type imageDataList []imageData

func (e imageDataList) Len() int {
	return len(e)
}

func (e imageDataList) Less(i, j int) bool {
	return e[i].width > e[j].width
}

func (e imageDataList) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func generateOutput(comparisonOperator compareType, compareValue int, fullPath string, height int, width int) imageData {
	var returnValue imageData

	if OrEqual {
		if (comparisonOperator == widerthan && width >= compareValue) ||
			(comparisonOperator == narrowerthan && width <= compareValue) ||
			(comparisonOperator == tallerthan && height >= compareValue) ||
			(comparisonOperator == shorterthan && height <= compareValue) {
			returnValue = imageData{name: fullPath, width: width, height: height}
		}
	} else {
		if (comparisonOperator == widerthan && width > compareValue) ||
			(comparisonOperator == narrowerthan && width < compareValue) ||
			(comparisonOperator == tallerthan && height > compareValue) ||
			(comparisonOperator == shorterthan && height < compareValue) {
			returnValue = imageData{name: fullPath, width: width, height: height}
		}
	}

	return returnValue
}

func scanFile(file fs.DirEntry, fileScans chan int, outputChannel chan<- imageData, waitGroup *sync.WaitGroup, comparisonOperator compareType, compareValue int, directory string) {
	defer func() {
		<-fileScans
		waitGroup.Done()
	}()

	fileScans <- 1

	fileName := file.Name()
	fullPath := filepath.Join(directory, fileName)

	reader, err := os.Open(fullPath)
	if err != nil {
		panic(err)
	}

	defer func() {
		err := reader.Close()
		if err != nil {
			panic(err)
		}
	}()

	myImage, _, err := image.DecodeConfig(reader)
	if errors.Is(err, image.ErrFormat) == true {
		return
	} else if err != nil {
		panic(err)
	}

	width := myImage.Width
	height := myImage.Height

	output := generateOutput(comparisonOperator, compareValue, fullPath, width, height)

	if output.name != "" {
		outputChannel <- output
	}
}

func scanDirectory(directoryScans chan int, fileScans chan int, outputChannel chan imageData, waitGroup *sync.WaitGroup, comparisonOperator compareType, compareValue int, directory string) {
	defer func() {
		<-directoryScans
		waitGroup.Done()
	}()

	directoryScans <- 1

	files, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}

	if len(files) == 0 {
		return
	}

	for _, file := range files {
		waitGroup.Add(1)

		go scanFile(file, fileScans, outputChannel, waitGroup, comparisonOperator, compareValue, directory)
	}
}

func scanDirectories(directoryScans chan int, fileScans chan int, outputChannel chan imageData, waitGroup *sync.WaitGroup, comparisonOperator compareType, compareValue int, arguments []string, dir int) {
	defer waitGroup.Done()

	if Recursive {
		directory := arguments[dir]

		filesystem := os.DirFS(directory)

		fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				waitGroup.Add(1)
				fullPath := filepath.Join(directory, path)
				go scanDirectory(directoryScans, fileScans, outputChannel, waitGroup, comparisonOperator, compareValue, fullPath)
			}

			return nil
		})
	} else {
		scanDirectory(directoryScans, fileScans, outputChannel, waitGroup, comparisonOperator, compareValue, arguments[dir])
	}
}

func ImageSizes(comparisonOperator compareType, arguments []string) {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		panic(err)
	}

	outputChannel := make(chan imageData)

	var waitGroup sync.WaitGroup

	directoryScans := make(chan int, maxDirectoryScans)
	fileScans := make(chan int, maxFileScans)

	for dir := 1; dir < len(arguments); dir++ {
		waitGroup.Add(1)
		go scanDirectories(directoryScans, fileScans, outputChannel, &waitGroup, comparisonOperator, compareValue, arguments, dir)
	}

	go func() {
		waitGroup.Wait()
		close(outputChannel)
		close(directoryScans)
		close(fileScans)
	}()

	var outputs []imageData

	for r := range outputChannel {
		outputs = append(outputs, r)
	}

	if !Unsorted {
		if SortOrder == "ascending" {
			if SortBy == height {
				sort.SliceStable(outputs, func(p, q int) bool {
					return outputs[p].height < outputs[q].height
				})
			} else if SortBy == width {
				sort.SliceStable(outputs, func(p, q int) bool {
					return outputs[p].width < outputs[q].width
				})
			} else {
				sort.SliceStable(outputs, func(p, q int) bool {
					return outputs[p].name < outputs[q].name
				})
			}
		} else {
			if SortBy == height {
				sort.SliceStable(outputs, func(p, q int) bool {
					return outputs[p].height > outputs[q].height
				})
			} else if SortBy == width {
				sort.SliceStable(outputs, func(p, q int) bool {
					return outputs[p].width > outputs[q].width
				})
			} else {
				sort.SliceStable(outputs, func(p, q int) bool {
					return outputs[p].name > outputs[q].name
				})
			}
		}
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
}
