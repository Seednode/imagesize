/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

type maxConcurrency int

const (
	// avoid hitting default open file descriptor limits (1024)
	maxDirectoryScans maxConcurrency = 32
	maxFileScans      maxConcurrency = 256

	Version string = "0.5.0"
)

var (
	count     bool
	orEqual   bool
	quiet     bool
	recursive bool
	sortOrder string
	sortBy    string
	unsorted  bool
	verbose   bool
	version   bool
)

var rootCmd = &cobra.Command{
	Use:              "imagesize",
	Short:            "displays images matching the specified constraints",
	TraverseChildren: true,
	Version:          Version,
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&count, "count", "c", false, "display number of matching files")
	rootCmd.PersistentFlags().BoolVar(&orEqual, "or-equal", false, "also match files equal to the specified dimension")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "silence filename output")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "include subdirectories")
	rootCmd.PersistentFlags().StringVar(&sortBy, "sort-by", "name", "sort output by the specified key")
	rootCmd.PersistentFlags().StringVar(&sortOrder, "sort-order", "ascending", "sort output in the specified direction")
	rootCmd.PersistentFlags().BoolVarP(&unsorted, "unsorted", "u", false, "do not sort output")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "display image dimensions in output")
	rootCmd.PersistentFlags().BoolVarP(&version, "version", "V", false, "display version and exit")

	rootCmd.SilenceErrors = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	rootCmd.SetVersionTemplate("imagesize v{{.Version}}\n")
	rootCmd.Version = Version
}
