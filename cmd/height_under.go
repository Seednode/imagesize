/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"log"

	"github.com/spf13/cobra"
)

var heightUnderCmd = &cobra.Command{
	Use:   "under <size in pixels> <directory1> [directory2]...",
	Short: "Filter images by height",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		err := ImageSizes(shorter, args)
		if err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	heightCmd.AddCommand(heightUnderCmd)
}
