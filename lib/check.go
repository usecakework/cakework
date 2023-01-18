package main

// should be i main?

import (
	"fmt"
	"os"
)

// TODO remove this; only the cli should os.exit
func CheckOsExit(e error) {
	if e != nil {
		fmt.Println(e)
		os.Exit(1)
		// TODO how to cause the program to exit?
	}
}