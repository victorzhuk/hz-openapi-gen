// Command hz-openapi-gen generates CloudWeGo Hertz server scaffold code
// (router, handler stubs, model structs) directly from an OpenAPI 3.x document.
//
// OpenAPI is the source of truth; the generator does not route the spec through
// hz, thriftgo, or protoc.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/diag"
	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/generator"
	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/openapi"
	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/writer"
)

// Exit codes (per spec).
const (
	exitOK          = 0
	exitError       = 1 // parse / validation / generation error
	exitUsage       = 2 // CLI usage error
	exitUnsafeWrite = 3 // refused to overwrite a non-generated file
)

var version = "dev"

type config struct {
	specPath           string
	outDir             string
	module             string
	packageBase        string
	dryRun             bool
	validateOnly       bool
	generateMain       bool
	generateGoGenerate bool
	force              bool
	strict             bool
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hz-openapi-gen", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var cfg config
	var showVersion bool
	fs.StringVar(&cfg.specPath, "spec", "", "path to the OpenAPI 3.x YAML/JSON file (required)")
	fs.StringVar(&cfg.outDir, "out", ".", "output root directory for generated files")
	fs.StringVar(&cfg.module, "module", "", "Go module path of the target service (required unless -validate-only)")
	fs.StringVar(&cfg.packageBase, "package-base", "biz", "base package folder for generated code")
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "print the generation plan without writing files")
	fs.BoolVar(&cfg.validateOnly, "validate-only", false, "validate the spec without generating files")
	fs.BoolVar(&cfg.generateMain, "generate-main", false, "generate a minimal main.go if missing")
	fs.BoolVar(&cfg.generateGoGenerate, "generate-go-generate", true, "generate generate.go with a go:generate directive")
	fs.BoolVar(&cfg.force, "force", false, "allow overwriting non-generated files")
	fs.BoolVar(&showVersion, "version", false, "print version and exit")
	fs.BoolVar(&cfg.strict, "strict", true, "treat missing operationIds and unsupported constructs as errors")
	fs.Usage = func() { usage(stderr, fs) }

	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if showVersion {
		fmt.Fprintln(stdout, "hz-openapi-gen "+version)
		return exitOK
	}
	if cfg.specPath == "" {
		fmt.Fprintln(stderr, "hz-openapi-gen: -spec is required")
		usage(stderr, fs)
		return exitUsage
	}
	if cfg.module == "" && !cfg.validateOnly {
		fmt.Fprintln(stderr, "hz-openapi-gen: -module is required (unless -validate-only)")
		usage(stderr, fs)
		return exitUsage
	}

	data, err := os.ReadFile(cfg.specPath)
	if err != nil {
		fmt.Fprintf(stderr, "hz-openapi-gen: read spec: %v\n", err)
		return exitError
	}

	doc, err := openapi.Load(data)
	if err != nil {
		fmt.Fprintf(stderr, "hz-openapi-gen: %v\n", err)
		return exitError
	}

	var diags diag.Set
	spec := openapi.Extract(doc, openapi.ExtractOptions{Strict: cfg.strict}, &diags)

	if cfg.validateOnly {
		diags.Report(stderr)
		if diags.HasErrors() {
			return exitError
		}
		fmt.Fprintf(stdout, "validation passed: %d operations, %d schemas, %d warnings\n",
			len(spec.Operations), len(spec.Models), len(diags.Warnings()))
		return exitOK
	}

	diags.Report(stderr)
	if diags.HasErrors() {
		return exitError
	}

	files, err := generator.Generate(spec, generator.Options{
		Module:             cfg.module,
		PackageBase:        cfg.packageBase,
		SpecPath:           cfg.specPath,
		GenerateMain:       cfg.generateMain,
		GenerateGoGenerate: cfg.generateGoGenerate,
	})
	if err != nil {
		fmt.Fprintf(stderr, "hz-openapi-gen: %v\n", err)
		return exitError
	}

	res, err := writer.Write(files, writer.Options{OutDir: cfg.outDir, DryRun: cfg.dryRun, Force: cfg.force})
	if err != nil {
		fmt.Fprintf(stderr, "hz-openapi-gen: %v\n", err)
		if errors.Is(err, writer.ErrUnsafeOverwrite) {
			return exitUnsafeWrite
		}
		return exitError
	}

	reportResult(stdout, cfg, spec, res)
	return exitOK
}

func reportResult(stdout io.Writer, cfg config, spec openapi.SpecModel, res writer.Result) {
	fmt.Fprintf(stdout, "hz-openapi-gen: loaded %s\n", cfg.specPath)
	fmt.Fprintf(stdout, "hz-openapi-gen: extracted %d operations, %d schemas\n", len(spec.Operations), len(spec.Models))

	if cfg.dryRun {
		fmt.Fprintln(stdout, "hz-openapi-gen: dry-run (no files written)")
		fmt.Fprintln(stdout, "planned files:")
		for _, p := range res.Added {
			fmt.Fprintf(stdout, "  + %s\n", p)
		}
		for _, p := range res.Changed {
			fmt.Fprintf(stdout, "  ~ %s\n", p)
		}
		for _, p := range res.Unchanged {
			fmt.Fprintf(stdout, "  = %s\n", p)
		}
		for _, p := range res.Skipped {
			fmt.Fprintf(stdout, "  (skip) %s\n", p)
		}
		return
	}

	for _, p := range res.Added {
		fmt.Fprintf(stdout, "hz-openapi-gen: wrote %s\n", p)
	}
	for _, p := range res.Changed {
		fmt.Fprintf(stdout, "hz-openapi-gen: updated %s\n", p)
	}
	for _, p := range res.Skipped {
		fmt.Fprintf(stdout, "hz-openapi-gen: skipped %s (exists, not generated)\n", p)
	}
	fmt.Fprintln(stdout, "hz-openapi-gen: done")
}

func usage(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprintln(w, "hz-openapi-gen generates CloudWeGo Hertz scaffold code from an OpenAPI 3.x spec.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  hz-openapi-gen -spec=api/openapi.yaml -out=. -module=example.com/service")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fs.PrintDefaults()
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  hz-openapi-gen -spec=api/openapi.yaml -validate-only")
	fmt.Fprintln(w, "  hz-openapi-gen -spec=api/openapi.yaml -out=. -module=example.com/service -dry-run")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Exit codes: 0 ok, 1 parse/validation/generation error, 2 usage error, 3 unsafe overwrite prevented")
}
