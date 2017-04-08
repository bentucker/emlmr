//go:generate goversioninfo -icon=emlmr.ico

package main

import (
	"fmt"
	"os"

	"github.com/bentucker/emlmr/cmd"
	flags "github.com/jessevdk/go-flags"
)

var version string
var options cmd.Options
var parser = flags.NewParser(&options, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	if options.Version {
		cmdName := parser.Name
		fmt.Fprintf(os.Stderr, "%s version %s\n", cmdName, version)
		os.Exit(0)
	}

	if len(options.Args.Files) == 0 {
		fmt.Println("No files specified\n")
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	if options.ListFields {
		cmd.ListFields(&options)
	} else {
		cmd.RunReport(&options)
	}
}
