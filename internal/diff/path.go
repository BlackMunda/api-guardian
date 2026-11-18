package diff

import "regexp"

var templateParam = regexp.MustCompile(`\{([^}]+)\}`)

// normalizePath replaces every {name} with {} so that /users/{id} and
// /users/{userId} count as the same endpoint. Path parameter names never
// appear on the wire (the request is "/users/123" either way), so renaming
// one is not a breaking change.
func normalizePath(p string) string { return templateParam.ReplaceAllString(p, "{}") }

// templateNames returns the parameter names in a path template, in order.
func templateNames(p string) []string {
	var names []string
	for _, m := range templateParam.FindAllStringSubmatch(p, -1) {
		names = append(names, m[1])
	}
	return names
}
