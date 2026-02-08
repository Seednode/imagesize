/*
Copyright © 2026 Seednode <seednode@seedno.de>
*/

package main

import (
	"github.com/spf13/cobra"
)

var widthOverCmd = &cobra.Command{
	Use:   "over <size in pixels> [directory1] ...[directoryN]",
	Short: "Filter images by width",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := imageSizes(wider, args)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	widthCmd.AddCommand(widthOverCmd)
}
