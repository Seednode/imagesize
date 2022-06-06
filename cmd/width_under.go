/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var widthUnderCmd = &cobra.Command{
	Use:   "under <size in pixels> <directory1> [directory2]...",
	Short: "Filter images by width",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ImageSizes(narrowerthan, args)
	},
}

func init() {
	widthCmd.AddCommand(widthUnderCmd)
}
