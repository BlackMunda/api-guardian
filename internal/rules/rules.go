// Package rules decides which Changes are breaking.
//
// The whole policy lives in one table so it is easy to read, test, and later
// make configurable. Anything not in the table is treated as breaking: for a
// safety tool, failing closed is better than silently passing.
package rules

import (
	"guardian/internal/diff"
)

// Rule explains one classification.
type Rule struct {
	Severity diff.Severity
	Why      string
}

var table = map[diff.Kind]Rule{
	diff.EndpointRemoved:           {diff.Breaking, "Clients calling this endpoint will get 404/405."},
	diff.EndpointAdded:             {diff.Safe, "Existing clients never call it."},
	diff.ResponseRemoved:           {diff.Breaking, "Clients expecting this success status code will see a different one."},
	diff.ResponseFieldRemoved:      {diff.Breaking, "Clients reading this field will get undefined/null."},
	diff.ResponseFieldAdded:        {diff.Safe, "Clients ignore fields they don't know about."},
	diff.ResponseFieldMadeOptional: {diff.Breaking, "Clients that relied on the field always being present may crash."},
	diff.RequestFieldAddedRequired: {diff.Breaking, "Existing clients don't send it, so their requests will be rejected."},
	diff.RequestFieldAddedOptional: {diff.Safe, "Existing requests remain valid."},
	diff.RequestFieldMadeRequired:  {diff.Breaking, "Clients that omitted it will now be rejected."},
	diff.RequestBodyAddedRequired:  {diff.Breaking, "Clients that send no body will now be rejected."},
	diff.ParamAddedRequired:        {diff.Breaking, "Existing clients don't send it, so their requests will be rejected."},
	diff.ParamAddedOptional:        {diff.Safe, "Existing requests remain valid."},
	diff.ParamMadeRequired:         {diff.Breaking, "Clients that omitted it will now be rejected."},
	diff.FieldTypeChanged:          {diff.Breaking, "Clients parse/serialize the old type."},
}

// Lookup returns the rule for a kind, and whether one is defined.
func Lookup(k diff.Kind) (Rule, bool) {
	r, ok := table[k]
	return r, ok
}

// SeverityOf returns the severity for a kind (Breaking if unknown).
func SeverityOf(k diff.Kind) diff.Severity {
	if r, ok := table[k]; ok {
		return r.Severity
	}
	return diff.Breaking
}

// Classify sets Severity on every change in place and returns the slice.
func Classify(changes []diff.Change) []diff.Change {
	for i := range changes {
		changes[i].Severity = SeverityOf(changes[i].Kind)
	}
	return changes
}

// Count returns how many changes are breaking and how many are safe.
func Count(changes []diff.Change) (breaking, safe int) {
	for _, c := range changes {
		if c.Severity == diff.Breaking {
			breaking++
		} else {
			safe++
		}
	}
	return
}
