/*
Copyright © 2026 Seednode <seednode@seedno.de>
*/

package main

import (
	"github.com/spf13/cobra"
)

var heightUnderCmd = &cobra.Command{
	Use:   "under <size in pixels> [directory1] ...[directoryN]",
	Short: "Filter images by height",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := imageSizes(shorter, args)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	heightCmd.AddCommand(heightUnderCmd)
}
