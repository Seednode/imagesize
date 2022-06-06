/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"image"
	"testing"

	"github.com/pkg/errors"
)

func TestParseSortBy(t *testing.T) {
	possibleOptions := []string{"name", "height", "width", "invalid"}
	expectedResponses := []SortKey{name, height, width, invalidSortKey}

	for option := range possibleOptions {
		SortBy = possibleOptions[option]
		Expected := expectedResponses[option]
		Returned, _ := parseSortBy()
		if Returned != Expected {
			t.Errorf("ParseSortBy() returned %v, expected %v.\n", Returned, Expected)
			t.FailNow()
		}
	}
}

func TestParseSortOrder(t *testing.T) {
	possibleOptions := []string{"ascending", "descending", "invalid"}
	expectedResponses := []SortDirection{ascending, descending, invalidSortDirection}

	for option := range possibleOptions {
		SortOrder = possibleOptions[option]
		Expected := expectedResponses[option]
		Returned, _ := parseSortOrder()
		if Returned != Expected {
			t.Errorf("ParseSortOrder() returned %v, expected %v.\n", Returned, Expected)
			t.FailNow()
		}
	}
}

func TestDecodeImage(t *testing.T) {
	fullPath := "../test/image.png"
	comparisonOperator := widerthan
	compareValue := 1

	outputChannel := make(chan ImageData)
	defer close(outputChannel)

	filePtr, reader, err := readFile(fullPath)
	if err != nil {
		t.Errorf("Reading file %v exited with error %q, expected none.\n", fullPath, err)
		t.FailNow()
	}
	defer func() {
		err := filePtr.Close()
		if err != nil {
			t.Errorf("Closing file %v exited with error %q, expected none.\n", fullPath, err)
			t.FailNow()
		}
	}()

	err = decodeImage(fullPath, reader, outputChannel, comparisonOperator, compareValue)
	if err != nil && !errors.Is(err, image.ErrFormat) {
		t.Errorf("Expected error %q, received %q.\n", image.ErrFormat, err)
		t.FailNow()
	}
}

func ExampleImageSizes() {
	Count = true
	OrEqual = false
	Quiet = false
	Recursive = true
	SortOrder = "ascending"
	SortBy = "name"
	Unsorted = false
	Verbose = true

	testArguments := []string{"512", "../test"}
	ImageSizes(tallerthan, testArguments)
	// Output:
	// ../test/1024x1.jpg (1x1024)
	// ../test/1024x1.png (1x1024)
	// ../test/1024x1024.jpg (1x1024)
	// ../test/1024x1024.png (1x1024)
	// ../test/15000x1024.jpg (1024x15000)
	// ../test/15000x1024.png (1024x15000)
	// ../test/subdirectory/1024x1.jpg (1x1024)
	// ../test/subdirectory/1024x1.png (1x1024)
	// ../test/subdirectory/1024x1024.jpg (1x1024)
	// ../test/subdirectory/1024x1024.png (1x1024)
	// ../test/subdirectory/15000x1024.jpg (1024x15000)
	// ../test/subdirectory/15000x1024.png (1024x15000)
	//
	// 12 file(s) matched.
}
