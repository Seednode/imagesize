/*
Copyright © 2025 Seednode <seednode@seedno.de>
*/

package main

import (
	"github.com/spf13/cobra"
)

var heightCmd = &cobra.Command{
	Use:   "height",
	Short: "Filter images by height"}

func init() {
	rootCmd.AddCommand(heightCmd)
}
