// Package openapi loads an OpenAPI 3.x document with libopenapi and extracts a
// generator-ready SpecModel. It is the only package that depends on libopenapi.
package openapi

import (
	"fmt"

	"github.com/pb33f/libopenapi"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// Load parses YAML or JSON bytes into an OpenAPI 3.x model. A Swagger 2.0 or
// otherwise non-3.x document fails here, since BuildV3Model rejects it.
func Load(data []byte) (*v3.Document, error) {
	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, fmt.Errorf("parse openapi document: %w", err)
	}
	model, err := doc.BuildV3Model()
	if err != nil {
		return nil, fmt.Errorf("build openapi 3.x model: %w", err)
	}
	return &model.Model, nil
}
