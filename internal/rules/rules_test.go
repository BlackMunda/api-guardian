package rules_test

import (
	"testing"

	"guardian/internal/diff"
	"guardian/internal/rules"
)

// Every Kind the diff engine can emit must have an explicit rule, so adding a
// new Kind without deciding its severity fails the build.
func TestEveryKindHasARule(t *testing.T) {
	for _, k := range diff.AllKinds {
		if _, ok := rules.Lookup(k); !ok {
			t.Errorf("no rule defined for kind %q", k)
		}
	}
}

func TestUnknownKindFailsClosed(t *testing.T) {
	if got := rules.SeverityOf("something_new"); got != diff.Breaking {
		t.Errorf("unknown kinds should be Breaking, got %v", got)
	}
}

func TestClassifyAndCount(t *testing.T) {
	changes := rules.Classify([]diff.Change{
		{Kind: diff.EndpointRemoved},
		{Kind: diff.EndpointAdded},
		{Kind: diff.ParamAddedOptional},
	})
	b, s := rules.Count(changes)
	if b != 1 || s != 2 {
		t.Errorf("Count = (%d breaking, %d safe), want (1, 2)", b, s)
	}
}
