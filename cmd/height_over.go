/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var heightOverCmd = &cobra.Command{
	Use:   "over <size in pixels> [directory1] ...[directoryN]",
	Short: "Filter images by height",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		err := imageSizes(taller, args)
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	heightCmd.AddCommand(heightOverCmd)
}
