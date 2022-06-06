/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var widthOverCmd = &cobra.Command{
	Use:   "over <size in pixels> <directory1> [directory2]...",
	Short: "Filter images by width",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ImageSizes(widerthan, args)
	},
}

func init() {
	widthCmd.AddCommand(widthOverCmd)
}
