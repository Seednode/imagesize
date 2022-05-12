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
	"os"
	"path/filepath"
	"strconv"
)

func scanDirectory(compareType string, compareValue int, directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}

	if len(files) == 0 {
		fmt.Println("Directory " + directory + " contains no files.")
	}

	for _, imageFile := range files {
		func() {
			fullPath := filepath.Join(directory, imageFile.Name())

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

			if OrEqual {
				if compareType == "wider-than" && myImage.Width >= compareValue {
					fmt.Println(fullPath)
				} else if compareType == "narrower-than" && myImage.Width <= compareValue {
					fmt.Println(fullPath)
				} else if compareType == "taller-than" && myImage.Height >= compareValue {
					fmt.Println(fullPath)
				} else if compareType == "shorter-than" && myImage.Height <= compareValue {
					fmt.Println(fullPath)
				}

				return
			}

			if compareType == "wider-than" && myImage.Width > compareValue {
				fmt.Println(fullPath)
			} else if compareType == "narrower-than" && myImage.Width < compareValue {
				fmt.Println(fullPath)
			} else if compareType == "taller-than" && myImage.Height > compareValue {
				fmt.Println(fullPath)
			} else if compareType == "shorter-than" && myImage.Height < compareValue {
				fmt.Println(fullPath)
			}
		}()
	}
}

func ImageSizes(compareType string, arguments []string) {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		panic(err)
	}

	for d := 1; d < len(arguments); d++ {
		scanDirectory(compareType, compareValue, arguments[d])
	}
}
