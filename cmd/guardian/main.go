// Guardian detects breaking changes between two versions of an OpenAPI spec.
//
//	guardian diff    [--format text|json|markdown] <base.yaml> <pr.yaml>
//	guardian comment [--name N] [--dry-run] [--repo o/r --pr N --token T] <base.yaml> <pr.yaml>
//
// Exit codes: 0 = no breaking changes, 1 = breaking changes found, 2 = error.
package main

import (
	"fmt"
	"io"
	"os"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	return runWith(args, stdout, stderr, os.Getenv)
}

// runWith takes getenv as a parameter so tests can fake the environment.
func runWith(args []string, stdout, stderr io.Writer, getenv func(string) string) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "diff":
		return runDiff(args[1:], stdout, stderr)
	case "comment":
		return runComment(args[1:], stdout, stderr, getenv)
	default:
		usage(stderr)
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "usage:")
	fmt.Fprintln(w, "  guardian diff    [--format text|json|markdown] <base.yaml> <pr.yaml>")
	fmt.Fprintln(w, "  guardian comment [--name N] [--dry-run] [--repo owner/name] [--pr N] [--token T] <base.yaml> <pr.yaml>")
}
