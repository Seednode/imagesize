/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var widthCmd = &cobra.Command{
	Use:   "width",
	Short: "Display all images over or under the specified width.",
}

func init() {
	rootCmd.AddCommand(widthCmd)
}
