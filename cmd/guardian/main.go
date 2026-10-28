// Guardian detects breaking changes between two versions of an OpenAPI spec.
//
//	guardian diff [--format text|json] base.yaml pr.yaml
//
// Exit codes: 0 = no breaking changes, 1 = breaking changes found, 2 = error.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"guardian/internal/check"
	"guardian/internal/report"
	"guardian/internal/rules"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	usage := func() {
		fmt.Fprintln(stderr, "usage: guardian diff [--format text|json] <base.yaml> <pr.yaml>")
	}
	if len(args) == 0 || args[0] != "diff" {
		usage()
		return 2
	}

	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if fs.NArg() != 2 || (*format != "text" && *format != "json") {
		usage()
		return 2
	}

	changes, err := check.Run(fs.Arg(0), fs.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	if *format == "json" {
		if err := report.JSON(stdout, changes); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 2
		}
	} else {
		report.Text(stdout, changes)
	}

	if breaking, _ := rules.Count(changes); breaking > 0 {
		return 1
	}
	return 0
}
