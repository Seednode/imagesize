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
	"strconv"
)

func scanDirectory(compareType string, compareValue int, directory string) {
	files, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}

	if len(files) == 0 {
		return
	}

	for _, file := range files {
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

		if Verbose && OrEqual {
			if compareType == "wider-than" && myImage.Width >= compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			} else if compareType == "narrower-than" && myImage.Width <= compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			} else if compareType == "taller-than" && myImage.Height >= compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			} else if compareType == "shorter-than" && myImage.Height <= compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			}
		} else if Verbose && !OrEqual {
			if compareType == "wider-than" && myImage.Width > compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			} else if compareType == "narrower-than" && myImage.Width < compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			} else if compareType == "taller-than" && myImage.Height > compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			} else if compareType == "shorter-than" && myImage.Height < compareValue {
				fmt.Printf("%v (%vx%v)\n", fullPath, myImage.Width, myImage.Height)
			}
		} else if !Verbose && OrEqual {
			if compareType == "wider-than" && myImage.Width >= compareValue {
				fmt.Println(fullPath)
			} else if compareType == "narrower-than" && myImage.Width <= compareValue {
				fmt.Println(fullPath)
			} else if compareType == "taller-than" && myImage.Height >= compareValue {
				fmt.Println(fullPath)
			} else if compareType == "shorter-than" && myImage.Height <= compareValue {
				fmt.Println(fullPath)
			}
		} else {
			if compareType == "wider-than" && myImage.Width > compareValue {
				fmt.Println(fullPath)
			} else if compareType == "narrower-than" && myImage.Width < compareValue {
				fmt.Println(fullPath)
			} else if compareType == "taller-than" && myImage.Height > compareValue {
				fmt.Println(fullPath)
			} else if compareType == "shorter-than" && myImage.Height < compareValue {
				fmt.Println(fullPath)
			}
		}
	}
}

func ImageSizes(compareType string, arguments []string) {
	compareValue, err := strconv.Atoi(arguments[0])
	if err != nil {
		panic(err)
	}

	for dir := 1; dir < len(arguments); dir++ {
		if Recursive {
			directory := arguments[dir]

			filesystem := os.DirFS(directory)

			fs.WalkDir(filesystem, ".", func(path string, d fs.DirEntry, err error) error {
				if d.IsDir() {
					fullPath := filepath.Join(directory, path)
					scanDirectory(compareType, compareValue, fullPath)
				}

				return nil
			})
		} else {
			scanDirectory(compareType, compareValue, arguments[dir])
		}
	}
}
