package main

import (
	"fmt"
	"os"
)

func main() {
	var err error
	if len(os.Args) > 1 {
		err = runCommand(os.Args[1:])
	} else {
		err = runConsole()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
