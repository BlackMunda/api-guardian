package diff

import (
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// jsonMedia picks the JSON media type from a content map, preferring
// application/json and falling back to any "...json" type (e.g. +json).
func jsonMedia(content openapi3.Content) *openapi3.MediaType {
	if mt := content.Get("application/json"); mt != nil {
		return mt
	}
	for _, k := range sortedKeys(content) {
		if strings.Contains(k, "json") {
			return content[k]
		}
	}
	return nil
}

// sortedKeys returns map keys in sorted order so output is deterministic.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
