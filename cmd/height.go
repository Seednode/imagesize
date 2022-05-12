/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var heightCmd = &cobra.Command{
	Use:   "height",
	Short: "Display all images over or under the specified height.",
}

func init() {
	rootCmd.AddCommand(heightCmd)
}
