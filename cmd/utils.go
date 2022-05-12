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

func scanFile(file fs.DirEntry, ch chan<- string, wg *sync.WaitGroup, compareType int, compareValue int, directory string) {
	defer wg.Done()

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

	var output string = ""

	if Verbose && OrEqual {
		if (compareType == widerthan && width >= compareValue) ||
			(compareType == narrowerthan && width <= compareValue) ||
			(compareType == tallerthan && height >= compareValue) ||
			(compareType == shorterthan && height <= compareValue) {
			output = fmt.Sprintf("%v (%dx%d)",
				fullPath, width, height)
		}
	} else if Verbose && !OrEqual {
		if (compareType == widerthan && width > compareValue) ||
			(compareType == narrowerthan && width < compareValue) ||
			(compareType == tallerthan && height > compareValue) ||
			(compareType == shorterthan && height < compareValue) {
			output = fmt.Sprintf("%v (%dx%d)",
				fullPath, width, height)
		}
	} else if !Verbose && OrEqual {
		if (compareType == widerthan && width >= compareValue) ||
			(compareType == narrowerthan && width <= compareValue) ||
			(compareType == tallerthan && height >= compareValue) ||
			(compareType == shorterthan && height <= compareValue) {
			output = fmt.Sprintf("%v", fullPath)
		}
	} else {
		if (compareType == widerthan && width > compareValue) ||
			(compareType == narrowerthan && width < compareValue) ||
			(compareType == tallerthan && height > compareValue) ||
			(compareType == shorterthan && height < compareValue) {
			output = fmt.Sprintf("%v", fullPath)
		}
	}

	if output != "" {
		ch <- output
	}
}

func scanDirectory(ch chan<- string, wg *sync.WaitGroup, compareType int, compareValue int, directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}

	if len(files) == 0 {
		return
	}

	for _, file := range files {
		wg.Add(1)

		go scanFile(file, ch, wg, compareType, compareValue, directory)
	}
}

func scanDirectories(ch chan<- string, wg *sync.WaitGroup, compareType int, compareValue int, arguments []string, dir int) {
	defer wg.Done()

	if Recursive {
		directory := arguments[dir]

		filesystem := os.DirFS(directory)

		fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				wg.Add(1)
				go func() {
					defer wg.Done()

					fullPath := filepath.Join(directory, path)
					scanDirectory(ch, wg, compareType, compareValue, fullPath)
				}()
			}

			return nil
		})
	} else {
		scanDirectory(ch, wg, compareType, compareValue, arguments[dir])
	}
}

func ImageSizes(compareType int, arguments []string) {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		panic(err)
	}

	ch := make(chan string)

	var wg sync.WaitGroup

	for dir := 1; dir < len(arguments); dir++ {
		wg.Add(1)

		go scanDirectories(ch, &wg, compareType, compareValue, arguments, dir)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var outputs []string

	for r := range ch {
		outputs = append(outputs, r)
	}

	sort.SliceStable(outputs, func(p, q int) bool {
		return outputs[p] < outputs[q]
	})

	if !Quiet {
		for o := 0; o < len(outputs); o++ {
			fmt.Printf("%v\n", outputs[o])
		}
	}

	if Count {
		fmt.Printf("\n%v file(s) matched.\n", len(outputs))
	}
}
