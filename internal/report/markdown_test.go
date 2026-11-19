package report

import (
	"fmt"
	"strings"
	"testing"

	"guardian/internal/diff"
)

func TestMarkdownBreaking(t *testing.T) {
	md := Markdown([]diff.Change{
		{Method: "GET", Path: "/users/{id}", Kind: diff.ResponseFieldRemoved,
			Message: "Response field 'email' was removed", Severity: diff.Breaking},
		{Method: "GET", Path: "/users", Kind: diff.ParamAddedOptional,
			Message: "New optional query parameter 'limit'", Severity: diff.Safe},
	})

	if !strings.HasPrefix(md, Marker+"\n") {
		t.Error("comment must start with the marker so it can be found and updated later")
	}
	for _, want := range []string{
		"❌ 1 breaking change\n",
		"| ❌ | `GET /users/{id}` | Response field 'email' was removed |",
		"Clients reading this field", // the "why" column comes from the rules table
		"<details>",
		"✅ 1 safe change",
		"| `GET /users` | New optional query parameter 'limit' |",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("markdown missing %q\n%s", want, md)
		}
	}
}

func TestMarkdownOnlySafe(t *testing.T) {
	md := Markdown([]diff.Change{{Method: "GET", Path: "/x", Message: "new", Severity: diff.Safe}})
	if !strings.Contains(md, "✅ no breaking changes") || strings.Contains(md, "Why it breaks") {
		t.Errorf("unexpected output:\n%s", md)
	}
}

func TestMarkdownNoChanges(t *testing.T) {
	md := Markdown(nil)
	if !strings.Contains(md, "✅ no API changes") || strings.Contains(md, "<details>") {
		t.Errorf("unexpected output:\n%s", md)
	}
}

func TestMarkdownEscapesTableCells(t *testing.T) {
	md := Markdown([]diff.Change{{Method: "GET", Path: "/x", Message: "a|b\nc", Severity: diff.Breaking}})
	if !strings.Contains(md, `a\|b c`) {
		t.Errorf("pipes/newlines not escaped:\n%s", md)
	}
}

func TestMarkdownTruncatesLongReports(t *testing.T) {
	var changes []diff.Change
	for i := 0; i < 80; i++ {
		changes = append(changes, diff.Change{Method: "GET", Path: fmt.Sprintf("/p%d", i),
			Message: "gone", Severity: diff.Breaking})
	}
	md := Markdown(changes)
	if !strings.Contains(md, "…and 30 more breaking changes not shown") {
		t.Errorf("expected truncation notice:\n%s", md)
	}
	if len(md) > 60000 {
		t.Errorf("comment too large: %d bytes", len(md))
	}
}

func TestMarkerFor(t *testing.T) {
	if MarkerFor("") != Marker {
		t.Error("an empty name must give the plain marker")
	}
	if got := MarkerFor("demo-api"); got != "<!-- guardian-report:demo-api -->" {
		t.Errorf("got %q", got)
	}
	// Names must not be able to break out of the HTML comment.
	if got := MarkerFor("x --> <script>"); strings.Contains(got, "<script>") || strings.Count(got, "-->") != 1 {
		t.Errorf("unsafe marker: %q", got)
	}
	if got := MarkerFor("my api"); got != "<!-- guardian-report:my-api -->" {
		t.Errorf("spaces should become dashes: %q", got)
	}
}

// Two checks on one PR must never find each other's comment.
func TestMarkersOfDifferentChecksDoNotCollide(t *testing.T) {
	names := []string{"", "demo", "demo-api", "users"}
	for _, a := range names {
		for _, b := range names {
			if a != b && strings.Contains(MarkerFor(a), MarkerFor(b)) {
				t.Errorf("marker for %q contains marker for %q", a, b)
			}
		}
	}
}

func TestMarkdownForNamedCheck(t *testing.T) {
	md := MarkdownFor(nil, "demo-api")
	if !strings.HasPrefix(md, "<!-- guardian-report:demo-api -->\n") {
		t.Errorf("wrong marker:\n%s", md)
	}
	if !strings.Contains(md, "## 🛡️ Guardian API check (demo-api): ✅ no API changes") {
		t.Errorf("heading should show the name:\n%s", md)
	}
}
