//go:build !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "airpodsd only runs on linux")
	os.Exit(1)
}
