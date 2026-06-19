package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalSpec = "../../testdata/minimal.yaml"

func TestRunExitCodes(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"validate ok", []string{"-spec=" + minimalSpec, "-validate-only"}, exitOK},
		{"no spec is usage error", []string{"-module=example.com/svc"}, exitUsage},
		{"generate without module is usage error", []string{"-spec=" + minimalSpec}, exitUsage},
		{"unknown flag is usage error", []string{"-spec=" + minimalSpec, "-nope"}, exitUsage},
		{"version prints and exits ok", []string{"-version"}, exitOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := run(tc.args, &bytes.Buffer{}, &bytes.Buffer{})
			assert.Equal(t, tc.want, code)
		})
	}
}

func TestRunMissingOperationIDFailsStrict(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "noid.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.3\ninfo: {title: t, version: 1.0.0}\npaths:\n  /x:\n    get:\n      responses: {'200': {description: ok}}\n"), 0o644))

	code := run([]string{"-spec=" + spec, "-out=" + dir, "-module=example.com/svc"}, &bytes.Buffer{}, &bytes.Buffer{})
	assert.Equal(t, exitError, code)
}

func TestRunUnsafeOverwrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "biz", "router")
	require.NoError(t, os.MkdirAll(target, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(target, "register.go"), []byte("package router // hand written"), 0o644))

	code := run([]string{"-spec=" + minimalSpec, "-out=" + dir, "-module=example.com/svc"}, &bytes.Buffer{}, &bytes.Buffer{})
	assert.Equal(t, exitUnsafeWrite, code)
}

func TestRunPreservesGeneratedHandlerEdits(t *testing.T) {
	dir := t.TempDir()

	code := run([]string{"-spec=" + minimalSpec, "-out=" + dir, "-module=example.com/svc", "-generate-main"}, &bytes.Buffer{}, os.Stderr)
	require.Equal(t, exitOK, code)

	handlerPath := filepath.Join(dir, "biz", "handler", "users.go")
	b, err := os.ReadFile(handlerPath)
	require.NoError(t, err)
	content := string(b)
	assert.Contains(t, content, "func CreateUser(")

	modified := content + "\nfunc customKeep() string { return \"keep\" }\n"
	require.NoError(t, os.WriteFile(handlerPath, []byte(modified), 0o644))

	code = run([]string{"-spec=" + minimalSpec, "-out=" + dir, "-module=example.com/svc", "-generate-main", "-force"}, &bytes.Buffer{}, os.Stderr)
	require.Equal(t, exitOK, code)

	b, err = os.ReadFile(handlerPath)
	require.NoError(t, err)
	final := string(b)
	assert.Contains(t, final, "func customKeep()", "custom code must survive regeneration")
	assert.Equal(t, 1, strings.Count(final, "func CreateUser("), "CreateUser must not be duplicated")
}

// TestGeneratedServiceCompiles is the real proof: generate a service, then
// build and vet it. Needs the module cache / network, so it is skipped in -short.
func TestGeneratedServiceCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile gate in -short mode")
	}
	specs := map[string]string{
		"minimal":  minimalSpec,
		"petstore": "../../testdata/petstore.yaml",
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			code := run([]string{"-spec=" + spec, "-out=" + dir, "-module=example.com/service", "-generate-main"}, &bytes.Buffer{}, os.Stderr)
			require.Equal(t, exitOK, code)

			runGo(t, dir, "mod", "init", "example.com/service")
			runGo(t, dir, "mod", "tidy")
			runGo(t, dir, "build", "./...")
			runGo(t, dir, "vet", "./...")
		})
	}
}

func runGo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go %v failed:\n%s", args, out)
}
