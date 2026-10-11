package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	flag.Usage = usage
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "airpods: %s\n", err)
		if errors.Is(err, errUsage) {
			usage()
			os.Exit(2)
		}
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: airpods status\n       airpods noise off|anc|transparency|adaptive\n")
}
