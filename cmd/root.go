/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

const (
	Version string = "0.6.1"
)

var (
	concurrency int
	orEqual     bool
	recursive   bool
	key         string
	order       string
	verbose     bool
	version     bool
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
	rootCmd.PersistentFlags().IntVarP(&concurrency, "max-concurrency", "c", 4096, "maximum number of paths to scan at once")
	rootCmd.PersistentFlags().BoolVarP(&orEqual, "or-equal", "e", false, "also match files equal to the specified dimension")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "include subdirectories")
	rootCmd.PersistentFlags().StringVarP(&key, "sort-key", "k", "name", "sort output by the specified key (height, width, name)")
	rootCmd.PersistentFlags().StringVarP(&order, "sort-order", "o", "ascending", "sort output in the specified direction (asc[ending], desc[ending])")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "display image dimensions and total matched file count")
	rootCmd.PersistentFlags().BoolVarP(&version, "version", "V", false, "display version and exit")

	rootCmd.SilenceErrors = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	rootCmd.SetVersionTemplate("imagesize v{{.Version}}\n")
	rootCmd.Version = Version
}
