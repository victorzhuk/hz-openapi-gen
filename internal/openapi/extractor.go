package openapi

import (
	"regexp"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"

	"github.com/victorzhuk/hz-openapi-gen/internal/diag"
)

// ExtractOptions controls how strictly the spec is interpreted.
type ExtractOptions struct {
	Strict bool
}

const jsonMedia = "application/json"

// Extract walks a resolved OpenAPI 3.x model in source order and returns a
// generator-ready SpecModel, recording any problems in d.
func Extract(doc *v3.Document, opts ExtractOptions, d *diag.Set) SpecModel {
	spec := SpecModel{}
	if doc.Info != nil {
		spec.Title = doc.Info.Title
		spec.Version = doc.Info.Version
	}
	spec.Operations = extractOperations(doc, opts, d)
	spec.Models = extractModels(doc, d)
	return spec
}

func extractOperations(doc *v3.Document, opts ExtractOptions, d *diag.Set) []OpDef {
	var ops []OpDef
	if doc.Paths == nil || doc.Paths.PathItems == nil {
		return ops
	}
	seenFunc := map[string]string{}
	for path, item := range doc.Paths.PathItems.FromOldest() {
		for methodLabel, op := range item.GetOperations().FromOldest() {
			method := strings.ToUpper(methodLabel)
			loc := method + " " + path

			funcName, ok := handlerFuncName(op, method, path, opts, d, loc)
			if !ok {
				continue
			}
			if prev, dup := seenFunc[funcName]; dup {
				d.Errorf("DUPLICATE_HANDLER_NAME", loc, "handler %q already generated for %s; give this operation a unique operationId", funcName, prev)
				continue
			}
			seenFunc[funcName] = loc

			tag := "default"
			if len(op.Tags) > 0 && op.Tags[0] != "" {
				tag = op.Tags[0]
			}

			ops = append(ops, OpDef{
				OperationID:   op.OperationId,
				FuncName:      funcName,
				Method:        method,
				HertzMethod:   method,
				OpenAPIPath:   path,
				HertzPath:     hertzPath(path),
				Summary:       op.Summary,
				Description:   op.Description,
				Tag:           tag,
				HandlerFile:   snake(tag) + ".go",
				RequestModel:  requestModel(op, loc, d),
				ResponseModel: primaryResponseModel(op, loc, d),
				Parameters:    extractParams(op, loc, d),
				Responses:     extractResponses(op),
			})
		}
	}
	return ops
}

func handlerFuncName(op *v3.Operation, method, path string, opts ExtractOptions, d *diag.Set, loc string) (string, bool) {
	if op.OperationId != "" {
		return Exported(op.OperationId), true
	}
	if opts.Strict {
		d.Errorf("OPENAPI_OPERATION_ID_MISSING", loc, "operation has no operationId; add one or run with -strict=false to synthesize a name")
		return "", false
	}
	name := Exported(strings.ToLower(method) + "_" + path)
	d.Warnf("OPENAPI_OPERATION_ID_SYNTHESIZED", loc, "no operationId; synthesized handler name %q", name)
	return name, true
}

var pathParamRe = regexp.MustCompile(`\{([^/}]+)\}`)

// hertzPath rewrites OpenAPI path templates to Hertz syntax: {id} -> :id.
func hertzPath(p string) string {
	return pathParamRe.ReplaceAllString(p, ":$1")
}

func extractParams(op *v3.Operation, loc string, d *diag.Set) []ParamDef {
	var out []ParamDef
	for _, p := range op.Parameters {
		if p == nil {
			continue
		}
		goType := "string"
		schemaType := ""
		if p.Schema != nil {
			goType = goTypeForProxy(p.Schema, loc, d)
			if !p.Schema.IsReference() {
				if s := p.Schema.Schema(); s != nil {
					schemaType = firstType(s.Type)
				}
			}
		}
		out = append(out, ParamDef{
			Name:       p.Name,
			GoName:     paramGoName(p.Name),
			In:         p.In,
			Required:   p.Required != nil && *p.Required,
			SchemaType: schemaType,
			GoType:     goType,
			BindTag:    bindTag(p),
		})
	}
	return out
}

func bindTag(p *v3.Parameter) string {
	switch p.In {
	case "path":
		return `path:"` + p.Name + `"`
	case "query":
		return `query:"` + p.Name + `"`
	case "header":
		return `header:"` + p.Name + `"`
	case "cookie":
		return `cookie:"` + p.Name + `"`
	default:
		return ""
	}
}

// requestModel returns the Go model name bound from a JSON request body, or ""
// when there is no JSON body. Inline (non-$ref) bodies are not modeled here.
func requestModel(op *v3.Operation, loc string, d *diag.Set) string {
	if op.RequestBody == nil || op.RequestBody.Content == nil {
		return ""
	}
	mt := op.RequestBody.Content.GetOrZero(jsonMedia)
	if mt == nil || mt.Schema == nil {
		return ""
	}
	if mt.Schema.IsReference() {
		return Exported(refName(mt.Schema.GetReference()))
	}
	d.Warnf("REQUEST_BODY_INLINE_UNSUPPORTED", loc, "inline request body schema is not supported; handler will not bind a typed request")
	return ""
}

// primaryResponseModel returns the model named by the lowest 2xx JSON response.
func primaryResponseModel(op *v3.Operation, _ string, _ *diag.Set) string {
	if op.Responses == nil || op.Responses.Codes == nil {
		return ""
	}
	var codes []string
	for code := range op.Responses.Codes.FromOldest() {
		codes = append(codes, code)
	}
	sort.Strings(codes)
	for _, code := range codes {
		if len(code) != 3 || code[0] != '2' {
			continue
		}
		resp := op.Responses.Codes.GetOrZero(code)
		if resp == nil || resp.Content == nil {
			continue
		}
		mt := resp.Content.GetOrZero(jsonMedia)
		if mt == nil || mt.Schema == nil {
			continue
		}
		if mt.Schema.IsReference() {
			return Exported(refName(mt.Schema.GetReference()))
		}
		return ""
	}
	return ""
}

func extractResponses(op *v3.Operation) []ResponseDef {
	var out []ResponseDef
	if op.Responses == nil || op.Responses.Codes == nil {
		return out
	}
	for code, resp := range op.Responses.Codes.FromOldest() {
		rd := ResponseDef{StatusCode: code}
		if resp != nil {
			rd.Description = resp.Description
			if resp.Content != nil {
				if mt := resp.Content.GetOrZero(jsonMedia); mt != nil {
					rd.MediaType = jsonMedia
					if mt.Schema != nil && mt.Schema.IsReference() {
						rd.ModelName = Exported(refName(mt.Schema.GetReference()))
					}
				}
			}
		}
		out = append(out, rd)
	}
	return out
}

func extractModels(doc *v3.Document, d *diag.Set) []ModelDef {
	var out []ModelDef
	if doc.Components == nil || doc.Components.Schemas == nil {
		return out
	}
	seen := map[string]string{}
	for name, sp := range doc.Components.Schemas.FromOldest() {
		goName := Exported(name)
		if prev, dup := seen[goName]; dup {
			d.Errorf("DUPLICATE_MODEL_NAME", name, "model %q collides with schema %q after sanitization", goName, prev)
			continue
		}
		seen[goName] = name

		md := ModelDef{
			Name:        goName,
			PackageName: "model",
			SourceRef:   "#/components/schemas/" + name,
		}
		sch := sp.Schema()
		if sch == nil {
			d.Warnf(codeSchemaBuild, name, "could not resolve schema (%v); generating empty struct", sp.GetBuildError())
			out = append(out, md)
			continue
		}
		md.Description = sch.Description
		md.Fields = extractFields(sch, name, d)
		out = append(out, md)
	}
	return out
}

func extractFields(sch *base.Schema, modelName string, d *diag.Set) []FieldDef {
	var fields []FieldDef
	if sch.Properties == nil {
		return fields
	}
	required := map[string]bool{}
	for _, r := range sch.Required {
		required[r] = true
	}
	for propName, propSchema := range sch.Properties.FromOldest() {
		loc := modelName + "." + propName
		f := FieldDef{
			Name:     fieldName(propName),
			JSONName: propName,
			GoType:   goTypeForProxy(propSchema, loc, d),
			Required: required[propName],
		}
		if propSchema != nil && !propSchema.IsReference() {
			if ps := propSchema.Schema(); ps != nil {
				f.Description = ps.Description
			}
		}
		fields = append(fields, f)
	}
	return fields
}
