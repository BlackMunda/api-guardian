package diff

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// Parameter comparison: path / query / header / cookie parameters.

type param struct {
	Name, In string
	Required bool
	Type     string
}

// collectParams merges path-level and operation-level parameters. Path
// parameters are keyed by their *position* in the URL template rather than
// their name, so a rename like {id} -> {userId} is not seen as remove + add.
func collectParams(path string, item *openapi3.PathItem, op *openapi3.Operation) map[string]param {
	names := templateNames(path)
	out := map[string]param{}
	add := func(list openapi3.Parameters) {
		for _, ref := range list {
			if ref == nil || ref.Value == nil {
				continue
			}
			v := ref.Value
			key := v.In + ":" + v.Name
			if v.In == "path" {
				for i, n := range names {
					if n == v.Name {
						key = fmt.Sprintf("path#%d", i)
					}
				}
			}
			t := ""
			if v.Schema != nil && v.Schema.Value != nil {
				t = typeOf(v.Schema.Value)
			}
			out[key] = param{Name: v.Name, In: v.In, Required: v.Required || v.In == "path", Type: t}
		}
	}
	add(item.Parameters)
	add(op.Parameters) // operation-level overrides path-level
	return out
}

func compareParams(c *collector, b, p opEntry) {
	bp := collectParams(b.Path, b.Item, b.Op)
	pp := collectParams(p.Path, p.Item, p.Op)
	for _, key := range sortedKeys(pp) {
		np := pp[key]
		loc := fmt.Sprintf("parameter.%s.%s", np.In, np.Name)
		old, existed := bp[key]
		if !existed {
			if np.Required {
				c.add(ParamAddedRequired, loc, "New required %s parameter '%s'", np.In, np.Name)
			} else {
				c.add(ParamAddedOptional, loc, "New optional %s parameter '%s'", np.In, np.Name)
			}
			continue
		}
		if !old.Required && np.Required {
			c.add(ParamMadeRequired, loc, "%s parameter '%s' is now required", np.In, np.Name)
		}
		if old.Type != "" && np.Type != "" && old.Type != np.Type {
			c.add(FieldTypeChanged, loc, "%s parameter '%s' type changed from %s to %s",
				np.In, np.Name, old.Type, np.Type)
		}
	}
}
