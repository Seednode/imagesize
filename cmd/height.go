/*
Copyright © 2024 Seednode <seednode@seedno.de>
*/

package cmd

import (
	"github.com/spf13/cobra"
)

var heightCmd = &cobra.Command{
	Use:   "height",
	Short: "Filter images by height"}

func init() {
	rootCmd.AddCommand(heightCmd)
}
