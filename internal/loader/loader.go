// Package loader reads an OpenAPI file from disk into kin-openapi's model.
// kin-openapi resolves $ref pointers for us, so later stages can follow
// schema.Value without worrying about references.
package loader

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

func Load(path string) (*openapi3.T, error) {
	l := openapi3.NewLoader()
	l.IsExternalRefsAllowed = true // allow $ref to other local files

	doc, err := l.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", path, err)
	}
	if err := doc.Validate(context.Background()); err != nil {
		return nil, fmt.Errorf("%s is not a valid OpenAPI document: %w", path, err)
	}
	return doc, nil
}
