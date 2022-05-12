/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var width_overCmd = &cobra.Command{
	Use:   "over <size in pixels> <directory1> [directory2]...",
	Short: "Display all images over the specified width.",
	Long:  "Display all images over the specified width in the directory or directories provided.",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		ImageSizes("wider-than", args)
	},
}

func init() {
	widthCmd.AddCommand(width_overCmd)
}
