/*
Copyright © 2022 Seednode <seednode@seedno.de>
*/

package main

import (
	"log"

	"seedno.de/seednode/imagesize/cmd"
)

func main() {
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
