package report

import (
	"fmt"
	"strings"

	"guardian/internal/diff"
	"guardian/internal/rules"
)

// Marker is a hidden HTML comment placed at the top of every Guardian PR
// comment. On later pushes we search for it so we can UPDATE the existing
// comment instead of posting a new one each time.
const Marker = "<!-- guardian-report -->"

// MarkerFor returns the marker for a named check. A repository can run
// several Guardian checks on one PR (say, two APIs); giving each its own name
// gives each its own comment, so they do not overwrite one another.
// An empty name returns the plain Marker.
func MarkerFor(name string) string {
	name = sanitizeName(name)
	if name == "" {
		return Marker
	}
	return "<!-- guardian-report:" + name + " -->"
}

// sanitizeName keeps only letters, digits, '.', '_' and '-' (spaces become
// '-'). This guarantees a name can never break out of the HTML comment.
func sanitizeName(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '_', r == '-':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// maxRows keeps the comment well below GitHub's 65,536-character limit.
const maxRows = 50

// Markdown renders the report as a GitHub-flavoured Markdown PR comment.
// Breaking changes are shown in a table with the reason; safe changes are
// tucked into a collapsed <details> block.
func Markdown(changes []diff.Change) string { return MarkdownFor(changes, "") }

// MarkdownFor is Markdown for a named check (see MarkerFor).
func MarkdownFor(changes []diff.Change, name string) string {
	name = sanitizeName(name)
	var breaking, safe []diff.Change
	for _, c := range changes {
		if c.Severity == diff.Breaking {
			breaking = append(breaking, c)
		} else {
			safe = append(safe, c)
		}
	}

	var b strings.Builder
	b.WriteString(MarkerFor(name) + "\n## 🛡️ Guardian API check")
	if name != "" {
		fmt.Fprintf(&b, " (%s)", name)
	}
	b.WriteString(": ")
	switch {
	case len(breaking) > 0:
		fmt.Fprintf(&b, "❌ %s\n\n", plural(len(breaking), "breaking change"))
	case len(safe) > 0:
		b.WriteString("✅ no breaking changes\n\n")
	default:
		b.WriteString("✅ no API changes\n\n")
	}

	if len(breaking) > 0 {
		b.WriteString("| | Endpoint | Change | Why it breaks |\n|---|---|---|---|\n")
		for i, c := range breaking {
			if i == maxRows {
				break
			}
			why := ""
			if r, ok := rules.Lookup(c.Kind); ok {
				why = r.Why
			}
			fmt.Fprintf(&b, "| ❌ | `%s %s` | %s | %s |\n", c.Method, c.Path, cell(c.Message), cell(why))
		}
		if len(breaking) > maxRows {
			fmt.Fprintf(&b, "\n_…and %d more breaking changes not shown._\n", len(breaking)-maxRows)
		}
		b.WriteString("\n")
	}

	if len(safe) > 0 {
		fmt.Fprintf(&b, "<details>\n<summary>✅ %s</summary>\n\n", plural(len(safe), "safe change"))
		b.WriteString("| Endpoint | Change |\n|---|---|\n")
		for i, c := range safe {
			if i == maxRows {
				break
			}
			fmt.Fprintf(&b, "| `%s %s` | %s |\n", c.Method, c.Path, cell(c.Message))
		}
		if len(safe) > maxRows {
			fmt.Fprintf(&b, "\n_…and %d more safe changes not shown._\n", len(safe)-maxRows)
		}
		b.WriteString("\n</details>\n\n")
	}

	b.WriteString("<sub>Posted by Guardian. This comment is updated on every push.</sub>\n")
	return b.String()
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// cell makes text safe inside a Markdown table cell.
func cell(s string) string {
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.ReplaceAll(s, "\n", " ")
}
