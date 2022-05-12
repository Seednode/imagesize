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

func scanDirectory(ch chan<- string, wg *sync.WaitGroup, compareType string, compareValue int, directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}

	if len(files) == 0 {
		return
	}

	for _, file := range files {
		wg.Add(1)
		go func(file fs.DirEntry) {
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
				if (compareType == "wider-than" && width >= compareValue) ||
					(compareType == "narrower-than" && width <= compareValue) ||
					(compareType == "taller-than" && height >= compareValue) ||
					(compareType == "shorter-than" && height <= compareValue) {
					output = fmt.Sprintf("%v (%dx%d)",
						fullPath, width, height)
				}
			} else if Verbose && !OrEqual {
				if (compareType == "wider-than" && width > compareValue) ||
					(compareType == "narrower-than" && width < compareValue) ||
					(compareType == "taller-than" && height > compareValue) ||
					(compareType == "shorter-than" && height < compareValue) {
					output = fmt.Sprintf("%v (%dx%d)",
						fullPath, width, height)
				}
			} else if !Verbose && OrEqual {
				if (compareType == "wider-than" && width >= compareValue) ||
					(compareType == "narrower-than" && width <= compareValue) ||
					(compareType == "taller-than" && height >= compareValue) ||
					(compareType == "shorter-than" && height <= compareValue) {
					output = fmt.Sprintf("%v", fullPath)
				}
			} else {
				if (compareType == "wider-than" && width > compareValue) ||
					(compareType == "narrower-than" && width < compareValue) ||
					(compareType == "taller-than" && height > compareValue) ||
					(compareType == "shorter-than" && height < compareValue) {
					output = fmt.Sprintf("%v", fullPath)
				}
			}

			if output != "" {
				ch <- output
			}
		}(file)
	}
}

func ImageSizes(compareType string, arguments []string) {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		panic(err)
	}

	ch := make(chan string)

	var wg sync.WaitGroup

	for dir := 1; dir < len(arguments); dir++ {
		wg.Add(1)

		a := dir
		go func() {
			defer wg.Done()

			if Recursive {
				directory := arguments[a]

				filesystem := os.DirFS(directory)

				fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
					if d.IsDir() {
						wg.Add(1)
						go func() {
							defer wg.Done()

							fullPath := filepath.Join(directory, path)
							scanDirectory(ch, &wg, compareType, compareValue, fullPath)
						}()
					}

					return nil
				})
			} else {
				scanDirectory(ch, &wg, compareType, compareValue, arguments[a])
			}
		}()
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

	for o := 0; o < len(outputs); o++ {
		fmt.Printf("%v\n", outputs[o])
	}
}
