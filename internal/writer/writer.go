package writer

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/victorzhuk/hz-openapi-gen/internal/generator"
	"github.com/victorzhuk/hz-openapi-gen/internal/hzutil"
)

var ErrUnsafeOverwrite = errors.New("unsafe overwrite prevented")

type Options struct {
	OutDir string
	DryRun bool
	Force  bool
}

type Result struct {
	Added     []string
	Changed   []string
	Unchanged []string
	Skipped   []string
	Warnings  []string
}

type plannedWrite struct {
	abs     string
	content []byte
}

func Write(files []generator.GeneratedFile, opts Options) (Result, error) {
	var res Result
	var writes []plannedWrite

	for _, f := range files {
		abs := filepath.Join(opts.OutDir, filepath.FromSlash(f.Path))
		existing, err := os.ReadFile(abs) //nolint:gosec // path from configured output directory
		exists := err == nil

		switch f.Mode {
		case generator.WriteCover:
			if exists && !opts.Force && !hasMarker(existing, generator.DoNotEditMarker) && !hasMarker(existing, generator.GeneratedMarker) {
				return res, fmt.Errorf("%w: %s does not contain %q", ErrUnsafeOverwrite, f.Path, generator.DoNotEditMarker)
			}
			if exists && bytes.Equal(existing, f.Content) {
				res.Unchanged = append(res.Unchanged, f.Path)
				continue
			}
			if exists {
				res.Changed = append(res.Changed, f.Path)
			} else {
				res.Added = append(res.Added, f.Path)
			}
			writes = append(writes, plannedWrite{abs: abs, content: f.Content})

		case generator.WriteCreate:
			if exists {
				res.Skipped = append(res.Skipped, f.Path)
				continue
			}
			res.Added = append(res.Added, f.Path)
			writes = append(writes, plannedWrite{abs: abs, content: f.Content})

		case generator.WriteMerge:
			if !exists {
				res.Added = append(res.Added, f.Path)
				writes = append(writes, plannedWrite{abs: abs, content: f.Content})
				continue
			}
			if !opts.Force && !hasMarker(existing, generator.GeneratedMarker) {
				return res, fmt.Errorf("%w: %s does not contain %q", ErrUnsafeOverwrite, f.Path, generator.GeneratedMarker)
			}
			if f.Merge == nil {
				return res, fmt.Errorf("merge plan missing for %s", f.Path)
			}
			for _, fn := range f.Merge.Functions {
				if hzutil.HasGoFunction(existing, fn.Name) && funcHasDelegateMarker(existing, fn.Name) {
					res.Warnings = append(res.Warnings, fmt.Sprintf("handler %s: existing function %s was generated in delegate mode and was not replaced", f.Path, fn.Name))
				}
			}
			merged, changed, err := mergeGoFile(existing, f.Merge)
			if err != nil {
				return res, fmt.Errorf("merge %s: %w", f.Path, err)
			}
			if !changed {
				res.Unchanged = append(res.Unchanged, f.Path)
				continue
			}
			res.Changed = append(res.Changed, f.Path)
			writes = append(writes, plannedWrite{abs: abs, content: merged})
		}
	}

	if !opts.DryRun {
		for _, w := range writes {
			if err := os.MkdirAll(filepath.Dir(w.abs), 0o755); err != nil { //nolint:gosec // wide permissions required for generated output directory
				return res, fmt.Errorf("create dir for %s: %w", w.abs, err)
			}
			if err := os.WriteFile(w.abs, w.content, 0o644); err != nil { //nolint:gosec // generated files are not executable
				return res, fmt.Errorf("write %s: %w", w.abs, err)
			}
		}
	}

	sort.Strings(res.Added)
	sort.Strings(res.Changed)
	sort.Strings(res.Unchanged)
	sort.Strings(res.Skipped)
	sort.Strings(res.Warnings)
	return res, nil
}

func hasMarker(content []byte, marker string) bool {
	head := content
	if nl := bytes.IndexByte(content, '\n'); nl >= 0 {
		head = content[:nl]
	}
	return bytes.Contains(head, []byte(marker))
}

func funcHasDelegateMarker(src []byte, name string) bool {
	start := bytes.Index(src, []byte("func "+name+"("))
	if start < 0 {
		return false
	}
	depth := 0
	for i := start; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return bytes.Contains(src[start:i+1], []byte("// delegated"))
			}
		}
	}
	return false
}

func mergeGoFile(existing []byte, plan *generator.GoMergePlan) (result []byte, changed bool, err error) {
	if plan == nil {
		return nil, false, errors.New("merge plan is nil")
	}

	merged := make([]byte, 0, len(existing)+4096)
	merged = append(merged, existing...)

	for _, fn := range plan.Functions {
		if hzutil.HasGoFunction(merged, fn.Name) {
			continue
		}

		for _, imp := range fn.Imports {
			if bytes.Contains(merged, []byte(fmt.Sprintf("%q", imp.Path))) {
				continue
			}
			var addErr error
			merged, addErr = hzutil.AddImportForContent(merged, imp.Alias, imp.Path)
			if addErr != nil {
				return nil, false, fmt.Errorf("add import %q for %s: %w", imp.Path, fn.Name, addErr)
			}
		}

		if len(merged) > 0 && merged[len(merged)-1] != '\n' {
			merged = append(merged, '\n')
		}
		merged = append(merged, '\n')
		merged = append(merged, fn.Content...)
		if len(fn.Content) > 0 && fn.Content[len(fn.Content)-1] != '\n' {
			merged = append(merged, '\n')
		}

		changed = true
	}

	if !changed {
		return nil, false, nil
	}

	formatted, fmtErr := hzutil.FormatGo("handler.go", merged)
	if fmtErr != nil {
		return nil, false, fmt.Errorf("format merged handler: %w", fmtErr)
	}

	return formatted, true, nil
}
