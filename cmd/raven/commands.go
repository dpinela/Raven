package main

import "fmt"

func runCommand(args []string) error {
	switch (args[0]) {
	case "setup":
		return setup(args[1:])
	case "install":
		return install(args[1:])
	case "list":
		return list(args[1:])
	case "yeet":
		return yeet(args[1:])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}
