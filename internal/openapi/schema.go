package openapi

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"

	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/diag"
)

const (
	codeOneOfUnsupported = "SCHEMA_ONE_OF_UNSUPPORTED"
	codeInlineObject     = "SCHEMA_INLINE_OBJECT_UNSUPPORTED"
	codeSchemaBuild      = "SCHEMA_BUILD_ERROR"
)

// goTypeForProxy maps a schema reference into a Go type. A $ref becomes the
// referenced model's name; everything else resolves through goTypeForSchema.
func goTypeForProxy(sp *base.SchemaProxy, loc string, d *diag.Set) string {
	if sp == nil {
		return "interface{}"
	}
	if sp.IsReference() {
		return Exported(refName(sp.GetReference()))
	}
	sch := sp.Schema()
	if sch == nil {
		d.Warnf(codeSchemaBuild, loc, "could not resolve schema (%v); using interface{}", sp.GetBuildError())
		return "interface{}"
	}
	return goTypeForSchema(sch, loc, d)
}

// goTypeForSchema applies the spec's OpenAPI-schema -> Go-type mapping table.
func goTypeForSchema(sch *base.Schema, loc string, d *diag.Set) string {
	switch {
	case len(sch.OneOf) > 0 || len(sch.AnyOf) > 0:
		d.Warnf(codeOneOfUnsupported, loc, "oneOf/anyOf is not supported in this prototype; using interface{}")
		return "interface{}"
	case len(sch.AllOf) > 0:
		d.Warnf(codeOneOfUnsupported, loc, "inline allOf is not supported as a field type; using interface{}")
		return "interface{}"
	}

	switch firstType(sch.Type) {
	case "string":
		switch sch.Format {
		case "date-time":
			return "time.Time"
		case "byte":
			return "[]byte"
		default:
			return "string"
		}
	case "integer":
		switch sch.Format {
		case "int32":
			return "int32"
		case "int64":
			return "int64"
		default:
			return "int"
		}
	case "number":
		if sch.Format == "float" {
			return "float32"
		}
		return "float64"
	case "boolean":
		return "bool"
	case "array":
		elem := "interface{}"
		if sch.Items != nil && sch.Items.IsA() {
			elem = goTypeForProxy(sch.Items.A, loc, d)
		}
		return "[]" + elem
	case "object", "":
		if sch.AdditionalProperties != nil && sch.AdditionalProperties.IsA() {
			return "map[string]" + goTypeForProxy(sch.AdditionalProperties.A, loc, d)
		}
		if hasEntries(sch.Properties) {
			d.Warnf(codeInlineObject, loc, "inline object schema is not supported; using map[string]interface{}")
			return "map[string]interface{}"
		}
		return "interface{}"
	default:
		return "interface{}"
	}
}

// firstType returns the first non-null entry of an OpenAPI 3.1 type list.
func firstType(types []string) string {
	for _, t := range types {
		if t != "null" {
			return t
		}
	}
	return ""
}

func hasEntries[K comparable, V any](m *orderedmap.Map[K, V]) bool {
	if m == nil {
		return false
	}
	for range m.FromOldest() {
		return true
	}
	return false
}
