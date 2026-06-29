package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/victorzhuk/hz-openapi-gen/internal/diag"
	"github.com/victorzhuk/hz-openapi-gen/internal/openapi"
)

func generateMinimal(t *testing.T) []GeneratedFile {
	t.Helper()
	data, err := os.ReadFile("../../testdata/minimal.yaml")
	require.NoError(t, err)
	doc, err := openapi.Load(data)
	require.NoError(t, err)
	var d diag.Set
	spec := openapi.Extract(doc, openapi.ExtractOptions{Strict: true}, &d)
	require.False(t, d.HasErrors())
	files, err := Generate(spec, Options{
		Module:             "example.com/service",
		PackageBase:        "biz",
		SpecPath:           "testdata/minimal.yaml",
		GenerateMain:       true,
		GenerateGoGenerate: true,
	})
	require.NoError(t, err)
	return files
}

// TestGenerateGolden compares generated output to committed snapshots.
// Regenerate with: UPDATE_GOLDEN=1 go test ./internal/generator/...
func TestGenerateGolden(t *testing.T) {
	files := generateMinimal(t)
	require.Len(t, files, 9)

	const goldenDir = "../../testdata/golden"
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	for _, f := range files {
		gp := filepath.Join(goldenDir, filepath.FromSlash(f.Path))
		if update {
			require.NoError(t, os.MkdirAll(filepath.Dir(gp), 0o755))
			require.NoError(t, os.WriteFile(gp, f.Content, 0o644))
			continue
		}
		want, err := os.ReadFile(gp)
		require.NoError(t, err, "missing golden for %s (run with UPDATE_GOLDEN=1)", f.Path)
		assert.Equal(t, string(want), string(f.Content), "golden mismatch for %s", f.Path)
	}
}

func TestGeneratedFilesAreMarkedAndValid(t *testing.T) {
	files := generateMinimal(t)
	byPath := map[string]string{}
	for _, f := range files {
		byPath[f.Path] = string(f.Content)
	}

	router := byPath["biz/router/register.go"]
	assert.True(t, strings.HasPrefix(router, DoNotEditMarker), "router file must start with DoNotEditMarker")
	assert.Contains(t, router, "func GeneratedRegister")
	assert.Contains(t, router, `h.GET("/users/:id", handler.GetUserByID)`)
	assert.Contains(t, router, `h.POST("/users", handler.CreateUser)`)

	handlerHealth := byPath["biz/handler/health.go"]
	assert.True(t, strings.HasPrefix(handlerHealth, GeneratedMarker), "handler file must start with GeneratedMarker")
	assert.NotContains(t, handlerHealth, "DO NOT EDIT")
	assert.NotContains(t, handlerHealth, "/biz/model")

	handlerUsers := byPath["biz/handler/users.go"]
	assert.True(t, strings.HasPrefix(handlerUsers, GeneratedMarker), "handler file must start with GeneratedMarker")
	assert.NotContains(t, handlerUsers, "DO NOT EDIT")
	assert.Contains(t, handlerUsers, "/biz/model")
	assert.Contains(t, handlerUsers, "consts.StatusBadRequest")
	assert.Contains(t, handlerUsers, "consts.StatusNotImplemented")
}

func generateDelegateMinimal(t *testing.T) []GeneratedFile {
	t.Helper()
	data, err := os.ReadFile("../../testdata/minimal.yaml")
	require.NoError(t, err)
	doc, err := openapi.Load(data)
	require.NoError(t, err)
	var d diag.Set
	spec := openapi.Extract(doc, openapi.ExtractOptions{Strict: true}, &d)
	require.False(t, d.HasErrors())
	files, err := Generate(spec, Options{
		Module:             "example.com/service",
		PackageBase:        "biz",
		SpecPath:           "testdata/minimal.yaml",
		GenerateMain:       true,
		GenerateGoGenerate: true,
		HandlerMode:        HandlerModeDelegate,
		DelegateImport:     "example.com/service/internal/public",
		DelegatePackage:    "public",
		DelegateFunc:       "Serve",
	})
	require.NoError(t, err)
	return files
}

func TestGenerateDelegateGolden(t *testing.T) {
	files := generateDelegateMinimal(t)
	require.Len(t, files, 10)

	const goldenDir = "../../testdata/golden-delegate"
	update := os.Getenv("UPDATE_GOLDEN") == "1"
	for _, f := range files {
		gp := filepath.Join(goldenDir, filepath.FromSlash(f.Path))
		if update {
			require.NoError(t, os.MkdirAll(filepath.Dir(gp), 0o755))
			require.NoError(t, os.WriteFile(gp, f.Content, 0o644))
			continue
		}
		want, err := os.ReadFile(gp)
		require.NoError(t, err, "missing golden for %s (run with UPDATE_GOLDEN=1)", f.Path)
		assert.Equal(t, string(want), string(f.Content), "golden mismatch for %s", f.Path)
	}
}

func TestDelegateHandlersReferenceConstants(t *testing.T) {
	files := generateDelegateMinimal(t)

	var opidContent string
	opidFound := false
	handlerFiles := map[string]string{}
	for _, f := range files {
		if f.Kind == "opid" {
			opidContent = string(f.Content)
			opidFound = true
		}
		if f.Kind == "handler" {
			assert.Equal(t, WriteCover, f.Mode, "handler %s must use WriteCover mode", f.Path)
			handlerFiles[f.Path] = string(f.Content)
		}
	}

	require.True(t, opidFound, "opid constants file must be generated")
	assert.Regexp(t, `GetHealth\s*=\s*"getHealth"`, opidContent)
	assert.Regexp(t, `ListUsers\s*=\s*"listUsers"`, opidContent)
	assert.Regexp(t, `CreateUser\s*=\s*"createUser"`, opidContent)
	assert.Regexp(t, `GetUserByID\s*=\s*"getUserById"`, opidContent)

	operationIDs := []string{"getHealth", "listUsers", "createUser", "getUserById"}
	require.NotEmpty(t, handlerFiles)
	funcNameRe := regexp.MustCompile(`func\s+([A-Za-z_]\w*)\s*\(`)
	for path, content := range handlerFiles {
		assert.True(t, strings.HasPrefix(content, DoNotEditMarker), "handler %s must start with DoNotEditMarker", path)
		assert.NotContains(t, content, "StatusNotImplemented", "handler %s must not contain StatusNotImplemented", path)
		assert.NotContains(t, content, "BindAndValidate", "handler %s must not contain BindAndValidate", path)
		for _, oid := range operationIDs {
			assert.NotContains(t, content, `"`+oid+`"`, "handler %s must not contain raw operationId %q", path, oid)
		}
		assert.Contains(t, content, "public.Serve(ctx, c, opid.", "handler %s must delegate via public.Serve", path)

		funcNames := funcNameRe.FindAllStringSubmatch(content, -1)
		require.NotEmpty(t, funcNames, "handler %s must define at least one function", path)
		for _, m := range funcNames {
			assert.Contains(t, content, "opid."+m[1], "handler %s must reference opid constant for function %s", path, m[1])
		}
	}
}
