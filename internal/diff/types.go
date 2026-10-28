package diff

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// Shared internal types used by every file in this package.
// They are the "contract" between the people working on different files,
// so change them only by agreement.

// direction says whether we are comparing something clients SEND (request)
// or something clients READ (response). The same edit can be breaking in one
// direction and harmless in the other.
type direction int

const (
	request direction = iota
	response
)

// opKey identifies an endpoint: normalized path + HTTP method.
type opKey struct{ Path, Method string }

// opEntry is one operation as found in one document.
type opEntry struct {
	Path   string // path as written in this document
	Method string
	Item   *openapi3.PathItem
	Op     *openapi3.Operation
}

// collector gathers Changes for a single operation.
type collector struct {
	path, method string
	changes      []Change
}

func (c *collector) add(kind Kind, location, format string, args ...any) {
	c.changes = append(c.changes, Change{
		Path: c.path, Method: c.method, Location: location,
		Kind: kind, Message: fmt.Sprintf(format, args...),
	})
}
