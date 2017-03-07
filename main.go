package main

import (
	"fmt"
	"os"

	"emlmr/cmd"
	flags "github.com/jessevdk/go-flags"
)

var version string

type Options struct {
	Delimiter string `short:"d" long:"delimiter" value-name:"DELIM" description:"use DELIM instead of COMMA for field delimiter."`
	Digest string `long:"digest" choice:"md5" choice:"sha1" description:"compute message digest for each email."`
	ListFields bool `short:"l" long:"list-fields" description:"list all metadata fields found."`
	Field   []string `short:"f" long:"field" default:"all" value-name:"FIELD" description:"include FIELD in report."`
	Output string `short:"o" long:"output" value-name:"FILE" description:"write output to FILE instead of stdout."`
	Version bool `long:"version" description:"Show application version and exit."`
	Args struct {
		FILE []string
	}   `positional-args:"yes" required:"yes"`
}

var options Options
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

	if len(options.Args.FILE) == 0 {
		fmt.Println("No files specified\n")
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	cmd.RunReport(options.Args.FILE, options.Field)
}
