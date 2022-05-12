/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var width_underCmd = &cobra.Command{
	Use:   "under <size in pixels> <directory1> [directory2]...",
	Short: "Display all images under the specified width.",
	Long:  "Display all images under the specified width in the directory or directories provided.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ImageSizes(narrowerthan, args)
	},
}

func init() {
	widthCmd.AddCommand(width_underCmd)
}
