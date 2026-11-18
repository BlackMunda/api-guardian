package diff

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// field is what we remember about one property of a schema.
type field struct {
	Type     string
	Required bool
}

// flatten turns a (possibly nested, $ref'd, allOf'd) schema into a flat map
// from field path to field info. Example for an array of orders:
//
//	"[]"              -> object
//	"[].id"           -> integer (required)
//	"[].address.city" -> string
//	"[].items[].qty"  -> integer
//
// Flat paths make comparing two schemas a simple map-vs-map comparison.
func flatten(ref *openapi3.SchemaRef) map[string]field {
	out := map[string]field{}
	if ref == nil || ref.Value == nil {
		return out
	}
	visit(ref.Value, "", true, map[*openapi3.Schema]bool{}, out)
	return out
}

func visit(s *openapi3.Schema, path string, required bool, stack map[*openapi3.Schema]bool, out map[string]field) {
	if path != "" {
		out[path] = field{Type: typeOf(s), Required: required}
	}
	// Recursive schemas (Category -> children -> Category) would loop forever.
	// "stack" holds the schemas currently being expanded; seeing one again
	// means we've gone around a cycle, so stop.
	if stack[s] {
		return
	}
	stack[s] = true
	defer delete(stack, s)

	props, req := collectProps(s)
	for name, p := range props {
		if p == nil || p.Value == nil {
			continue
		}
		visit(p.Value, join(path, name), req[name], stack, out)
	}
	if s.Items != nil && s.Items.Value != nil {
		visit(s.Items.Value, path+"[]", true, stack, out)
	}
}

// collectProps gathers properties and required-lists, merging allOf members.
func collectProps(s *openapi3.Schema) (map[string]*openapi3.SchemaRef, map[string]bool) {
	props := map[string]*openapi3.SchemaRef{}
	req := map[string]bool{}
	seen := map[*openapi3.Schema]bool{}
	var add func(*openapi3.Schema)
	add = func(x *openapi3.Schema) {
		if seen[x] {
			return
		}
		seen[x] = true
		for n, p := range x.Properties {
			props[n] = p
		}
		for _, r := range x.Required {
			req[r] = true
		}
		for _, sub := range x.AllOf {
			if sub != nil && sub.Value != nil {
				add(sub.Value)
			}
		}
	}
	add(s)
	return props, req
}

func typeOf(s *openapi3.Schema) string {
	if s.Type != nil {
		if t := s.Type.Slice(); len(t) > 0 {
			return strings.Join(t, "|")
		}
	}
	switch {
	case len(s.Properties) > 0 || len(s.AllOf) > 0:
		return "object"
	case s.Items != nil:
		return "array"
	case len(s.OneOf) > 0 || len(s.AnyOf) > 0:
		return "oneOf/anyOf"
	}
	return "any"
}

func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// parentOf returns the path of the enclosing field: "a.b" -> "a", "a[]" -> "a",
// "a[].b" -> "a[]", "a" -> "".
func parentOf(path string) string {
	if strings.HasSuffix(path, "[]") {
		return strings.TrimSuffix(path, "[]")
	}
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[:i]
	}
	return ""
}
