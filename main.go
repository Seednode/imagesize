/*
Copyright © 2026 Seednode <seednode@seedno.de>
*/

package main

import (
	"log"

	"github.com/spf13/cobra"
)

const (
	ReleaseVersion string = "1.2.0"
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
}

func main() {
	rootCmd.PersistentFlags().IntVarP(&concurrency, "max-concurrency", "c", 4096, "maximum number of paths to scan at once")
	rootCmd.PersistentFlags().BoolVarP(&orEqual, "or-equal", "e", false, "also match files equal to the specified dimension")
	rootCmd.PersistentFlags().BoolVarP(&recursive, "recursive", "r", false, "include subdirectories")
	rootCmd.PersistentFlags().StringVarP(&key, "sort-key", "k", "name", "sort output by the specified key (height, width, name)")
	rootCmd.PersistentFlags().StringVarP(&order, "sort-order", "o", "ascending", "sort output in the specified direction (asc[ending], desc[ending])")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "display image dimensions and total matched file count")
	rootCmd.PersistentFlags().BoolVarP(&version, "version", "V", false, "display version and exit")

	rootCmd.Flags().SetInterspersed(true)

	rootCmd.CompletionOptions.HiddenDefaultCmd = true

	rootCmd.SilenceErrors = true
	rootCmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	rootCmd.SetVersionTemplate("imagesize v{{.Version}}\n")
	rootCmd.Version = ReleaseVersion

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
