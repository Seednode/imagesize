/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var Version string = "0.1"

var OrEqual bool
var Recursive bool
var Verbose bool

var rootCmd = &cobra.Command{
	Short:            "Displays images matching specified dimensional constraints.",
	TraverseChildren: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&OrEqual, "or-equal", false, "match files equal to the provided dimension")
	rootCmd.PersistentFlags().BoolVarP(&Recursive, "recursive", "r", false, "include subdirectories")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "display image dimensions in output")
}
