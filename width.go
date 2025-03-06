/*
Copyright © 2025 Seednode <seednode@seedno.de>
*/

package main

import (
	"github.com/spf13/cobra"
)

var widthCmd = &cobra.Command{
	Use:   "width",
	Short: "Filter images by width",
}

func init() {
	rootCmd.AddCommand(widthCmd)
}
