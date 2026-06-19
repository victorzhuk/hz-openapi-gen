package generator

import (
	"fmt"
	"sort"

	"github.com/victorzhuk/hz-openapi-gen/internal/hzutil"
	"github.com/victorzhuk/hz-openapi-gen/internal/openapi"
)

const generatorModule = "github.com/victorzhuk/hz-openapi-gen"

type WriteMode string

const (
	WriteCover  WriteMode = "cover"
	WriteCreate WriteMode = "create"
	WriteMerge  WriteMode = "merge"
)

type GoImport struct {
	Alias string
	Path  string
}

type GoFunction struct {
	Name    string
	Content []byte
	Imports []GoImport
}

type GoMergePlan struct {
	Functions []GoFunction
}

type GeneratedFile struct {
	Path    string
	Content []byte
	Kind    string
	Mode    WriteMode
	Merge   *GoMergePlan
}

type Options struct {
	Module             string
	PackageBase        string
	SpecPath           string
	GenerateMain       bool
	GenerateGoGenerate bool
}

var hertzVerb = map[string]bool{
	"GET": true, "POST": true, "PUT": true, "DELETE": true,
	"PATCH": true, "HEAD": true, "OPTIONS": true,
}

func routerStmt(method, hpath, fn string) string {
	if hertzVerb[method] {
		return fmt.Sprintf("h.%s(%q, handler.%s)", method, hpath, fn)
	}
	return fmt.Sprintf("h.Handle(%q, %q, handler.%s)", method, hpath, fn)
}

func Generate(spec openapi.SpecModel, opts Options) ([]GeneratedFile, error) {
	base := opts.PackageBase
	if base == "" {
		base = "biz"
	}

	var files []GeneratedFile

	routerFile, err := renderRouter(spec, opts, base)
	if err != nil {
		return nil, err
	}
	files = append(files, routerFile)

	handlerFiles, err := renderHandlers(spec, opts, base)
	if err != nil {
		return nil, err
	}
	files = append(files, handlerFiles...)

	modelFiles, err := renderModels(spec, base)
	if err != nil {
		return nil, err
	}
	files = append(files, modelFiles...)

	if opts.GenerateMain {
		content, err := render("main", map[string]any{
			"RouterImport": hzutil.SubPackage(opts.Module, hzutil.SubDir(base, "router")),
		})
		if err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{Path: "main.go", Content: content, Kind: "main", Mode: WriteCreate})
	}

	if opts.GenerateGoGenerate {
		content, err := render("generate", map[string]any{
			"GenModule": generatorModule,
			"SpecPath":  opts.SpecPath,
			"Module":    opts.Module,
		})
		if err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{Path: "generate.go", Content: content, Kind: "generate", Mode: WriteCreate})
	}

	return files, nil
}

func renderRouter(spec openapi.SpecModel, opts Options, base string) (GeneratedFile, error) {
	type routeLine struct{ Stmt string }
	routes := make([]routeLine, 0, len(spec.Operations))
	for i := range spec.Operations {
		op := &spec.Operations[i]
		routes = append(routes, routeLine{Stmt: routerStmt(op.HertzMethod, op.HertzPath, op.FuncName)})
	}
	handlerDir := hzutil.SubDir(base, "handler")
	content, err := render("router", map[string]any{
		"HandlerImport": hzutil.SubPackage(opts.Module, handlerDir),
		"Routes":        routes,
	})
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{
		Path:    hzutil.SubDir(hzutil.SubDir(base, "router"), "register.go"),
		Content: content,
		Kind:    "router",
		Mode:    WriteCover,
	}, nil
}

func renderHandlers(spec openapi.SpecModel, opts Options, base string) ([]GeneratedFile, error) {
	groups := map[string][]openapi.OpDef{}
	var order []string
	for i := range spec.Operations {
		op := &spec.Operations[i]
		if _, ok := groups[op.HandlerFile]; !ok {
			order = append(order, op.HandlerFile)
		}
		groups[op.HandlerFile] = append(groups[op.HandlerFile], *op)
	}
	sort.Strings(order)

	handlerDir := hzutil.SubDir(base, "handler")
	modelDir := hzutil.SubDir(base, "model")
	modelImport := hzutil.SubPackage(opts.Module, modelDir)

	var files []GeneratedFile
	for _, hf := range order {
		handlers := groups[hf]

		needModel := false
		for i := range handlers {
			if handlers[i].RequestModel != "" {
				needModel = true
				break
			}
		}

		content, err := render("handler", map[string]any{
			"ModelImport": modelImport,
			"NeedModel":   needModel,
			"Handlers":    handlers,
		})
		if err != nil {
			return nil, err
		}

		mergePlan := &GoMergePlan{}
		for i := range handlers {
			op := &handlers[i]
			funcContent, err := renderHandlerFunc(*op)
			if err != nil {
				return nil, fmt.Errorf("render handler function %s: %w", op.FuncName, err)
			}
			imports := []GoImport{
				{Alias: "", Path: "context"},
				{Alias: "", Path: "github.com/cloudwego/hertz/pkg/app"},
				{Alias: "", Path: "github.com/cloudwego/hertz/pkg/protocol/consts"},
			}
			if op.RequestModel != "" {
				imports = append(imports, GoImport{Alias: "model", Path: modelImport})
			}
			mergePlan.Functions = append(mergePlan.Functions, GoFunction{
				Name:    op.FuncName,
				Content: funcContent,
				Imports: imports,
			})
		}

		files = append(files, GeneratedFile{
			Path:    hzutil.SubDir(handlerDir, hf),
			Content: content,
			Kind:    "handler",
			Mode:    WriteMerge,
			Merge:   mergePlan,
		})
	}
	return files, nil
}

func renderModels(spec openapi.SpecModel, base string) ([]GeneratedFile, error) {
	modelDir := hzutil.SubDir(base, "model")
	var files []GeneratedFile
	for _, m := range spec.Models {
		content, err := render("model", map[string]any{
			"Model":    m,
			"NeedTime": m.UsesTime(),
		})
		if err != nil {
			return nil, err
		}
		files = append(files, GeneratedFile{
			Path:    hzutil.SubDir(modelDir, openapi.FileStem(m.Name)+".go"),
			Content: content,
			Kind:    "model",
			Mode:    WriteCover,
		})
	}
	return files, nil
}
