package diff

import "strings"

func location(scope, path string) string {
	if strings.HasPrefix(path, "[") {
		return scope + path
	}
	return scope + "." + path
}

// compareFields is where the direction matters. Clients *send* request fields
// and *read* response fields, so the same change has opposite effects:
//
//	new required field   : breaks request (clients don't send it), harmless in response
//	removed field        : breaks response (clients read it), harmless in request
//	required -> optional : breaks response (field may vanish), harmless in request
func compareFields(c *collector, scope string, dir direction, base, pr map[string]field) {
	for _, path := range sortedKeys(pr) {
		pf := pr[path]
		bf, existed := base[path]
		loc := location(scope, path)

		if !existed {
			// If the parent is new too, the parent's report already covers it.
			if parent := parentOf(path); parent != "" {
				if _, ok := base[parent]; !ok {
					continue
				}
			}
			switch {
			case dir == request && pf.Required:
				c.add(RequestFieldAddedRequired, loc, "Request field '%s' is now required", path)
			case dir == request:
				c.add(RequestFieldAddedOptional, loc, "New optional request field '%s'", path)
			default:
				c.add(ResponseFieldAdded, loc, "New response field '%s'", path)
			}
			continue
		}

		if bf.Type != pf.Type {
			c.add(FieldTypeChanged, loc, "Field '%s' type changed from %s to %s", path, bf.Type, pf.Type)
		}
		if dir == request && !bf.Required && pf.Required {
			c.add(RequestFieldMadeRequired, loc, "Request field '%s' is now required", path)
		}
		if dir == response && bf.Required && !pf.Required {
			c.add(ResponseFieldMadeOptional, loc, "Response field '%s' is no longer guaranteed (now optional)", path)
		}
	}

	if dir == response {
		for _, path := range sortedKeys(base) {
			if _, ok := pr[path]; ok {
				continue
			}
			if parent := parentOf(path); parent != "" {
				if _, ok := pr[parent]; !ok {
					continue
				}
			}
			c.add(ResponseFieldRemoved, location(scope, path), "Response field '%s' was removed", path)
		}
	}
}
