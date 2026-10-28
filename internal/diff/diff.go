package diff

import (
	"sort"

	"github.com/getkin/kin-openapi/openapi3"
)

// Compare walks two OpenAPI documents and returns every difference that could
// matter to an API consumer. Severity is left unset; see rules.Classify.
func Compare(base, pr *openapi3.T) []Change {
	baseOps := indexOperations(base)
	prOps := indexOperations(pr)

	var out []Change
	for key, b := range baseOps {
		p, ok := prOps[key]
		if !ok {
			out = append(out, Change{
				Path: b.Path, Method: b.Method, Location: "endpoint",
				Kind: EndpointRemoved, Message: "Endpoint was removed",
			})
			continue
		}
		out = append(out, compareOperation(b, p)...)
	}
	for key, p := range prOps {
		if _, ok := baseOps[key]; !ok {
			out = append(out, Change{
				Path: p.Path, Method: p.Method, Location: "endpoint",
				Kind: EndpointAdded, Message: "New endpoint",
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Method != b.Method {
			return a.Method < b.Method
		}
		if a.Location != b.Location {
			return a.Location < b.Location
		}
		return a.Kind < b.Kind
	})
	return out
}

func indexOperations(doc *openapi3.T) map[opKey]opEntry {
	idx := map[opKey]opEntry{}
	if doc == nil || doc.Paths == nil {
		return idx
	}
	for path, item := range doc.Paths.Map() {
		if item == nil {
			continue
		}
		for method, op := range item.Operations() {
			idx[opKey{normalizePath(path), method}] = opEntry{path, method, item, op}
		}
	}
	return idx
}

// compareOperation runs the three comparisons for one endpoint that exists in
// both documents: parameters, request body, responses.
func compareOperation(b, p opEntry) []Change {
	c := &collector{path: p.Path, method: p.Method}
	compareParams(c, b, p)
	compareRequestBody(c, b, p)
	compareResponses(c, b, p)
	return c.changes
}
