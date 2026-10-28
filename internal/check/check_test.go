package check_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"guardian/internal/check"
	"guardian/internal/diff"
)

// want describes one expected change. Each case in testdata/<name>/ must
// produce exactly this set of changes: no more, no fewer.
type want struct {
	Sev      diff.Severity
	Kind     diff.Kind
	Method   string
	Path     string
	Location string
}

func (w want) String() string {
	return fmt.Sprintf("%-8s %-30s %-6s %-14s %s", w.Sev, w.Kind, w.Method, w.Path, w.Location)
}

const (
	B = diff.Breaking
	S = diff.Safe
)

func TestTestdataCases(t *testing.T) {
	cases := []struct {
		dir  string
		want []want
	}{
		{"removed_field", []want{
			{B, diff.ResponseFieldRemoved, "GET", "/users/{id}", "response.200.email"},
		}},
		{"new_required_param", []want{
			{S, diff.ParamAddedOptional, "GET", "/users", "parameter.query.limit"},
			{B, diff.ParamAddedRequired, "GET", "/users", "parameter.query.tenant"},
			{B, diff.RequestFieldAddedRequired, "POST", "/orders", "request.body.currency"},
			{B, diff.RequestFieldMadeRequired, "POST", "/orders", "request.body.quantity"},
			{S, diff.RequestFieldAddedOptional, "POST", "/orders", "request.body.note"},
		}},
		{"type_changed", []want{
			{B, diff.FieldTypeChanged, "GET", "/users", "parameter.query.limit"},
			{B, diff.FieldTypeChanged, "GET", "/users", "response.200.id"},
			{B, diff.FieldTypeChanged, "POST", "/orders", "request.body.quantity"},
		}},
		{"endpoint_removed", []want{
			{B, diff.EndpointRemoved, "DELETE", "/users/{id}", "endpoint"},
			{B, diff.EndpointRemoved, "GET", "/orders", "endpoint"},
			{S, diff.EndpointAdded, "POST", "/users", "endpoint"},
		}},
		{"safe_additions", []want{
			{S, diff.EndpointAdded, "GET", "/products", "endpoint"},
			{S, diff.ParamAddedOptional, "GET", "/users", "parameter.query.page"},
			{S, diff.ResponseFieldAdded, "GET", "/users", "response.200.nickname"},
			{S, diff.ResponseFieldAdded, "GET", "/users", "response.200.tags"},
			{S, diff.RequestFieldAddedOptional, "POST", "/users", "request.body.bio"},
		}},
		{"nested_ref", []want{
			{B, diff.ResponseFieldRemoved, "GET", "/orders", "response.200[].address.city"},
			{B, diff.ResponseFieldRemoved, "GET", "/orders", "response.200[].createdAt"},
			{B, diff.ResponseFieldRemoved, "GET", "/orders", "response.200[].items[].qty"},
			{B, diff.ResponseFieldMadeOptional, "GET", "/orders", "response.200[].id"},
			{B, diff.FieldTypeChanged, "GET", "/orders", "response.200[].tags[]"},
		}},
		// Renaming {id} -> {userId} is invisible on the wire, so: no changes.
		{"path_param_renamed", nil},
		// The scenario we show in the demo.
		{"demo", []want{
			{B, diff.RequestFieldMadeRequired, "POST", "/orders", "request.body.currency"},
			{B, diff.ResponseFieldRemoved, "GET", "/users", "response.200[].email"},
			{B, diff.ResponseFieldRemoved, "GET", "/users/{id}", "response.200.email"},
			{S, diff.ParamAddedOptional, "GET", "/users", "parameter.query.limit"},
		}},
	}

	for _, tc := range cases {
		t.Run(tc.dir, func(t *testing.T) {
			dir := filepath.Join("..", "..", "testdata", tc.dir)
			changes, err := check.Run(filepath.Join(dir, "base.yaml"), filepath.Join(dir, "pr.yaml"))
			if err != nil {
				t.Fatal(err)
			}

			var got, exp []string
			for _, c := range changes {
				got = append(got, want{c.Severity, c.Kind, c.Method, c.Path, c.Location}.String())
			}
			for _, w := range tc.want {
				exp = append(exp, w.String())
			}
			sort.Strings(got)
			sort.Strings(exp)

			if !reflect.DeepEqual(got, exp) {
				t.Errorf("unexpected changes\n--- got ---\n%v\n--- want ---\n%v", join(got), join(exp))
			}
		})
	}
}

func join(lines []string) string {
	if len(lines) == 0 {
		return "  (none)"
	}
	s := ""
	for _, l := range lines {
		s += "  " + l + "\n"
	}
	return s
}

func TestInvalidFileIsAnError(t *testing.T) {
	if _, err := check.Run("does-not-exist.yaml", "also-missing.yaml"); err == nil {
		t.Fatal("expected an error for missing files")
	}
}
