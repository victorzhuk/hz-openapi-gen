package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/diag"
	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/openapi"
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
