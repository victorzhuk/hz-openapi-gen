package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/generator"
)

func marked(path, body string) generator.GeneratedFile {
	return generator.GeneratedFile{
		Path:    path,
		Content: []byte(generator.DoNotEditMarker + "\n\n" + body),
		Mode:    generator.WriteCover,
	}
}

func TestWriteCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	res, err := Write([]generator.GeneratedFile{marked("biz/x.go", "package biz\n")}, Options{OutDir: dir})
	require.NoError(t, err)
	assert.Equal(t, []string{"biz/x.go"}, res.Added)

	b, err := os.ReadFile(filepath.Join(dir, "biz", "x.go"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "package biz")
}

func TestDryRunWritesNothing(t *testing.T) {
	dir := t.TempDir()
	res, err := Write([]generator.GeneratedFile{marked("a.go", "package a\n")}, Options{OutDir: dir, DryRun: true})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.go"}, res.Added)

	_, statErr := os.Stat(filepath.Join(dir, "a.go"))
	assert.True(t, os.IsNotExist(statErr), "dry-run must not write files")
}

func TestRefusesUnmarkedOverwrite(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a // hand written"), 0o644))

	files := []generator.GeneratedFile{marked("a.go", "package a\n")}
	_, err := Write(files, Options{OutDir: dir})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnsafeOverwrite)

	// the file must be untouched after a refused write
	b, err := os.ReadFile(filepath.Join(dir, "a.go"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "hand written")

	// -force overrides
	_, err = Write(files, Options{OutDir: dir, Force: true})
	require.NoError(t, err)
}

func TestOverwritesMarkedFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte(generator.DoNotEditMarker+"\npackage a // old\n"), 0o644))

	res, err := Write([]generator.GeneratedFile{marked("a.go", "package a // new\n")}, Options{OutDir: dir})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.go"}, res.Changed)
}

func TestNoOpWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	f := marked("a.go", "package a\n")
	_, err := Write([]generator.GeneratedFile{f}, Options{OutDir: dir})
	require.NoError(t, err)

	res, err := Write([]generator.GeneratedFile{f}, Options{OutDir: dir})
	require.NoError(t, err)
	assert.Equal(t, []string{"a.go"}, res.Unchanged)
}

func TestOnceFileSkippedWhenExists(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\nfunc main(){}\n"), 0o644))

	once := generator.GeneratedFile{Path: "main.go", Content: []byte("package main // generated\n"), Mode: generator.WriteCreate}
	res, err := Write([]generator.GeneratedFile{once}, Options{OutDir: dir})
	require.NoError(t, err)
	assert.Equal(t, []string{"main.go"}, res.Skipped)

	b, err := os.ReadFile(filepath.Join(dir, "main.go"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "func main(){}", "a once-file must not clobber an existing file")
}

func TestWriteMergesGeneratedHandler(t *testing.T) {
	dir := t.TempDir()

	handlerContent := generator.GeneratedMarker + `

package handler

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

func GetHealth(ctx context.Context, c *app.RequestContext) {
	c.JSON(consts.StatusNotImplemented, map[string]string{"error": "not implemented"})
}

func customKeep() string {
	return "keep"
}
`
	handlerPath := filepath.Join(dir, "biz", "handler", "health.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(handlerPath), 0o755))
	require.NoError(t, os.WriteFile(handlerPath, []byte(handlerContent), 0o644))

	mergeFile := generator.GeneratedFile{
		Path: "biz/handler/health.go",
		Kind: "handler",
		Mode: generator.WriteMerge,
		Merge: &generator.GoMergePlan{
			Functions: []generator.GoFunction{
				{
					Name:    "GetHealth",
					Content: []byte("func GetHealth(ctx context.Context, c *app.RequestContext) {\n\tc.JSON(consts.StatusNotImplemented, map[string]string{\"error\": \"not implemented\"})\n}\n"),
					Imports: []generator.GoImport{
						{Alias: "", Path: "context"},
						{Alias: "", Path: "github.com/cloudwego/hertz/pkg/app"},
						{Alias: "", Path: "github.com/cloudwego/hertz/pkg/protocol/consts"},
					},
				},
				{
					Name:    "CreateUser",
					Content: []byte("func CreateUser(ctx context.Context, c *app.RequestContext) {\n\tvar req model.CreateUserRequest\n\tif err := c.BindAndValidate(&req); err != nil {\n\t\tc.JSON(consts.StatusBadRequest, map[string]string{\"error\": err.Error()})\n\t\treturn\n\t}\n\tc.JSON(consts.StatusNotImplemented, map[string]string{\"error\": \"not implemented\"})\n}\n"),
					Imports: []generator.GoImport{
						{Alias: "", Path: "context"},
						{Alias: "", Path: "github.com/cloudwego/hertz/pkg/app"},
						{Alias: "", Path: "github.com/cloudwego/hertz/pkg/protocol/consts"},
						{Alias: "model", Path: "example.com/biz/model"},
					},
				},
			},
		},
	}

	res, err := Write([]generator.GeneratedFile{mergeFile}, Options{OutDir: dir})
	require.NoError(t, err)
	assert.Equal(t, []string{"biz/handler/health.go"}, res.Changed)

	b, err := os.ReadFile(handlerPath)
	require.NoError(t, err)
	content := string(b)
	assert.Contains(t, content, "func customKeep()", "custom helper must survive merge")
	assert.Contains(t, content, "func CreateUser(", "missing handler must be appended")
	assert.Contains(t, content, `"example.com/biz/model"`, "model import must be added")
	assert.Equal(t, 1, countFunc(content, "GetHealth"), "existing handler must not be duplicated")
	assert.Equal(t, 1, countFunc(content, "CreateUser"), "new handler must appear exactly once")

	// second write must report Unchanged
	res2, err := Write([]generator.GeneratedFile{mergeFile}, Options{OutDir: dir})
	require.NoError(t, err)
	assert.Equal(t, []string{"biz/handler/health.go"}, res2.Unchanged)
}

func TestWriteRefusesUnmarkedMerge(t *testing.T) {
	dir := t.TempDir()

	handlerContent := `package handler

import "context"

func GetHealth(ctx context.Context, c *app.RequestContext) {}
`
	handlerPath := filepath.Join(dir, "biz", "handler", "users.go")
	require.NoError(t, os.MkdirAll(filepath.Dir(handlerPath), 0o755))
	require.NoError(t, os.WriteFile(handlerPath, []byte(handlerContent), 0o644))

	mergeFile := generator.GeneratedFile{
		Path: "biz/handler/users.go",
		Kind: "handler",
		Mode: generator.WriteMerge,
		Merge: &generator.GoMergePlan{
			Functions: []generator.GoFunction{
				{
					Name:    "CreateUser",
					Content: []byte("func CreateUser(ctx context.Context, c *app.RequestContext) {}\n"),
					Imports: []generator.GoImport{
						{Alias: "", Path: "context"},
						{Alias: "", Path: "github.com/cloudwego/hertz/pkg/app"},
					},
				},
			},
		},
	}

	_, err := Write([]generator.GeneratedFile{mergeFile}, Options{OutDir: dir})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrUnsafeOverwrite)

	// file must be untouched
	b, err := os.ReadFile(handlerPath)
	require.NoError(t, err)
	assert.Equal(t, handlerContent, string(b))
}

func countFunc(src, name string) int {
	n := 0
	for i := 0; i < len(src); {
		idx := strings.Index(src[i:], "func "+name+"(")
		if idx < 0 {
			break
		}
		n++
		i += idx + 5
	}
	return n
}
