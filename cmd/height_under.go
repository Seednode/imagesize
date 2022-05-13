/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var height_underCmd = &cobra.Command{
	Use:   "under <size in pixels> <directory1> [directory2]...",
	Short: "Filter images by height",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ImageSizes(shorterthan, args)
	},
}

func init() {
	heightCmd.AddCommand(height_underCmd)
}
