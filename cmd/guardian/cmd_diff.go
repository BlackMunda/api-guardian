package main

import (
	"flag"
	"fmt"
	"io"

	"guardian/internal/check"
	"guardian/internal/report"
	"guardian/internal/rules"
)

// runDiff: compare two specs and print the report. Exit 1 if anything breaks.
func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	format := fs.String("format", "text", "output format: text, json or markdown")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	valid := *format == "text" || *format == "json" || *format == "markdown"
	if fs.NArg() != 2 || !valid {
		usage(stderr)
		return 2
	}

	changes, err := check.Run(fs.Arg(0), fs.Arg(1))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	switch *format {
	case "json":
		if err := report.JSON(stdout, changes); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 2
		}
	case "markdown":
		fmt.Fprint(stdout, report.Markdown(changes))
	default:
		report.Text(stdout, changes)
	}

	if breaking, _ := rules.Count(changes); breaking > 0 {
		return 1
	}
	return 0
}
