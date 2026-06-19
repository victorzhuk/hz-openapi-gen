package openapi

// SpecModel is the fully resolved, generator-ready view of an OpenAPI document.
// Every Go type, name, and Hertz path is computed during extraction, so the
// generator package renders templates without touching libopenapi.
type SpecModel struct {
	Title      string
	Version    string
	Operations []OpDef
	Models     []ModelDef
}

// OpDef is one HTTP operation mapped onto a Hertz route + handler.
type OpDef struct {
	OperationID   string
	FuncName      string
	Method        string
	HertzMethod   string
	OpenAPIPath   string
	HertzPath     string
	Summary       string
	Description   string
	Tag           string
	HandlerFile   string
	RequestModel  string
	ResponseModel string
	Parameters    []ParamDef
	Responses     []ResponseDef
}

// ParamDef is a path/query/header/cookie parameter.
type ParamDef struct {
	Name       string
	GoName     string
	In         string
	Required   bool
	SchemaType string
	GoType     string
	BindTag    string
}

// ResponseDef is a single declared response.
type ResponseDef struct {
	StatusCode  string
	Description string
	ModelName   string
	MediaType   string
}

// ModelDef is a Go struct generated from a schema.
type ModelDef struct {
	Name        string
	PackageName string
	Fields      []FieldDef
	Description string
	SourceRef   string
}

// FieldDef is one struct field.
type FieldDef struct {
	Name           string
	JSONName       string
	GoType         string
	Required       bool
	Description    string
	ValidationTags []string
}

// UsesTime reports whether any field renders a time.Time, so the model file can
// import "time" only when needed.
func (m ModelDef) UsesTime() bool {
	for _, f := range m.Fields {
		if f.GoType == "time.Time" || f.GoType == "[]time.Time" || f.GoType == "*time.Time" {
			return true
		}
	}
	return false
}
