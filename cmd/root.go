/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
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

const (
	name   string = "name"
	height string = "height"
	width  string = "width"
)

var (
	Count     bool
	OrEqual   bool
	Quiet     bool
	Recursive bool
	SortBy    string
	Unsorted  bool
	Verbose   bool
	Version   string = "0.2"
)

var rootCmd = &cobra.Command{
	Use:              "imagesize",
	Short:            "Displays images matching the specified constraints.",
	TraverseChildren: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&Count, "count", "c", false, "display number of matching files")
	rootCmd.PersistentFlags().BoolVar(&OrEqual, "or-equal", false, "also match files equal to the provided dimension")
	rootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "silence filename output")
	rootCmd.PersistentFlags().BoolVarP(&Recursive, "recursive", "r", false, "include subdirectories")
	rootCmd.PersistentFlags().StringVarP(&SortBy, "sort-by", "s", "name", "sort output by the provided dimension")
	rootCmd.PersistentFlags().BoolVarP(&Unsorted, "unsorted", "u", false, "do not sort output")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "display image dimensions in output")
}
