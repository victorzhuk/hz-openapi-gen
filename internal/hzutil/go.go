package hzutil

import (
	"bytes"
	"fmt"
	"go/format"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"

	"golang.org/x/tools/go/ast/astutil"
)

func FormatGo(filename string, src []byte) ([]byte, error) {
	if filepath.Ext(filepath.Base(filename)) != ".go" {
		return src, nil
	}
	formatted, err := format.Source(src)
	if err != nil {
		return nil, fmt.Errorf("format generated %s: %w\n--- source ---\n%s", filename, err, src)
	}
	return formatted, nil
}

func AddImportForContent(fileContent []byte, alias, importPath string) ([]byte, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", fileContent, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if !astutil.AddNamedImport(fset, f, alias, importPath) {
		return nil, fmt.Errorf("add import %q: already present or failed", importPath)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return nil, fmt.Errorf("format after import: %w", err)
	}
	return buf.Bytes(), nil
}

func HasGoFunction(src []byte, name string) bool {
	quoted := regexp.QuoteMeta(name)
	pat := `func\s+(\([^)]*\)\s+)?` + quoted + `\s*\(`
	return regexp.MustCompile(pat).Match(src)
}
