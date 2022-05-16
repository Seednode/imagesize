/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

type sortOrder int

const (
	ascending sortOrder = iota
	descending
)

type sortBy int

const (
	name sortBy = iota
	height
	width
)

type compareType int

const (
	widerthan compareType = iota
	narrowerthan
	tallerthan
	shorterthan
)

type maxConcurrency int

const (
	// avoid hitting default open file descriptor limits (1024)
	maxDirectoryScans maxConcurrency = 32
	maxFileScans      maxConcurrency = 256
)

var (
	Count     bool
	OrEqual   bool
	Quiet     bool
	Recursive bool
	SortOrder string
	SortBy    string
	Unsorted  bool
	Verbose   bool
	Version   string = "0.2"
)

var rootCmd = &cobra.Command{
	Use:              "imagesize",
	Short:            "displays images matching the specified constraints",
	TraverseChildren: true,
	Version:          Version,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Count, "count", "c", false, "display number of matching files")
	rootCmd.PersistentFlags().BoolVar(&OrEqual, "or-equal", false, "also match files equal to the specified dimension")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "silence filename output")
	rootCmd.PersistentFlags().BoolVarP(&Recursive, "recursive", "r", false, "include subdirectories")
	rootCmd.PersistentFlags().StringVar(&SortBy, "sort-by", "name", "sort output by the specified key")
	rootCmd.PersistentFlags().StringVar(&SortOrder, "sort-order", "ascending", "sort output in the specified direction")
	rootCmd.PersistentFlags().BoolVarP(&Unsorted, "unsorted", "u", false, "do not sort output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "display image dimensions in output")
}
