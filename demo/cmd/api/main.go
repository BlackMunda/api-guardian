// Command api runs the demo API, or prints its OpenAPI spec.
//
//	go run ./cmd/api openapi      # print the OpenAPI 3.0 spec (YAML) to stdout
//	go run ./cmd/api serve [addr] # run the server (default :8888)
//
// The "openapi" command is what Guardian's CI runs on the base branch and on
// the pull request, to compare the two generated specs.
package main

import (
	"fmt"
	"net/http"
	"os"

	"guardian-demo/internal/api"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "openapi":
		spec, err := api.SpecYAML()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		os.Stdout.Write(spec)
	case "serve":
		addr := ":8888"
		if len(os.Args) > 2 {
			addr = os.Args[2]
		}
		mux := http.NewServeMux()
		api.New(mux)
		fmt.Fprintf(os.Stderr, "listening on %s (docs at /docs)\n", addr)
		if err := http.ListenAndServe(addr, mux); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() { fmt.Fprintln(os.Stderr, "usage: api openapi | api serve [addr]") }
