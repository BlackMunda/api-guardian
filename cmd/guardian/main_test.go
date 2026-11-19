package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
	}{
		{"breaking -> 1", []string{"diff", "../../testdata/demo/base.yaml", "../../testdata/demo/pr.yaml"}, 1},
		{"safe only -> 0", []string{"diff", "../../testdata/safe_additions/base.yaml", "../../testdata/safe_additions/pr.yaml"}, 0},
		{"no changes -> 0", []string{"diff", "../../testdata/path_param_renamed/base.yaml", "../../testdata/path_param_renamed/pr.yaml"}, 0},
		{"missing file -> 2", []string{"diff", "nope.yaml", "nope2.yaml"}, 2},
		{"bad usage -> 2", []string{"diff", "only-one-arg.yaml"}, 2},
		{"no command -> 2", nil, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var out, errb bytes.Buffer
			if got := run(tc.args, &out, &errb); got != tc.code {
				t.Errorf("exit code = %d, want %d\nstdout: %s\nstderr: %s", got, tc.code, out.String(), errb.String())
			}
		})
	}
}

func TestJSONOutput(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"diff", "--format", "json",
		"../../testdata/demo/base.yaml", "../../testdata/demo/pr.yaml"}, &out, &errb)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	for _, s := range []string{`"severity": "breaking"`, `"kind": "response_field_removed"`, `"breaking": 3`} {
		if !strings.Contains(out.String(), s) {
			t.Errorf("JSON output missing %s", s)
		}
	}
}

func TestMarkdownFormat(t *testing.T) {
	var out, errb bytes.Buffer
	code := run([]string{"diff", "--format", "markdown",
		"../../testdata/demo/base.yaml", "../../testdata/demo/pr.yaml"}, &out, &errb)
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.HasPrefix(out.String(), "<!-- guardian-report -->") || !strings.Contains(out.String(), "❌ 3 breaking changes") {
		t.Errorf("unexpected markdown:\n%s", out.String())
	}
}

func TestUnknownCommand(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"frobnicate"}, &out, &errb); code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
}
