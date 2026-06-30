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

const minimalSpec = "testdata/minimal.yaml"

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

func TestResolveVersion(t *testing.T) {
	orig := version
	t.Cleanup(func() { version = orig })

	t.Run("ldflag override wins", func(t *testing.T) {
		version = "v1.2.3"
		assert.Equal(t, "v1.2.3", resolveVersion())
	})

	t.Run("dev sentinel without release build info", func(t *testing.T) {
		version = "dev"
		assert.Equal(t, "dev", resolveVersion())
	})
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

func TestRunDelegateModeValidation(t *testing.T) {
	delegateImport := "-delegate-import=example.com/svc/internal/public"
	delegatePkg := "-delegate-package=public"
	delegateFunc := "-delegate-func=Serve"

	cases := []struct {
		name  string
		flags []string
		want  int
	}{
		{"stub default works without delegate flags", nil, exitOK},
		{"valid delegate mode works", []string{"-handler-mode=delegate", delegateImport, delegatePkg, delegateFunc}, exitOK},
		{"missing delegate-import is usage error", []string{"-handler-mode=delegate", delegatePkg, delegateFunc}, exitUsage},
		{"missing delegate-func is usage error", []string{"-handler-mode=delegate", delegateImport, delegatePkg}, exitUsage},
		{"invalid delegate-package is usage error", []string{"-handler-mode=delegate", delegateImport, "-delegate-package=123bad", delegateFunc}, exitUsage},
		{"invalid delegate-func is usage error", []string{"-handler-mode=delegate", delegateImport, delegatePkg, "-delegate-func=123bad"}, exitUsage},
		{"reserved package name is usage error", []string{"-handler-mode=delegate", delegateImport, "-delegate-package=context", delegateFunc}, exitUsage},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			args := []string{"-spec=" + minimalSpec, "-out=" + dir, "-module=example.com/svc"}
			args = append(args, tc.flags...)
			code := run(args, &bytes.Buffer{}, &bytes.Buffer{})
			assert.Equal(t, tc.want, code)
		})
	}
}

func TestRunDelegateModeRequiresOperationID(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "noid.yaml")
	require.NoError(t, os.WriteFile(spec, []byte("openapi: 3.0.3\ninfo: {title: t, version: 1.0.0}\npaths:\n  /x:\n    get:\n      responses: {'200': {description: ok}}\n"), 0o644))

	var stderr bytes.Buffer
	code := run([]string{
		"-spec=" + spec,
		"-out=" + dir,
		"-module=example.com/svc",
		"-strict=false",
		"-handler-mode=delegate",
		"-delegate-import=example.com/svc/internal/public",
		"-delegate-package=public",
		"-delegate-func=Serve",
	}, &bytes.Buffer{}, &stderr)
	assert.Equal(t, exitError, code)
	assert.Contains(t, stderr.String(), "delegate mode requires operationId")
}

// TestGeneratedServiceCompiles is the real proof: generate a service, then
// build and vet it. Needs the module cache / network, so it is skipped in -short.
func TestGeneratedServiceCompiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping compile gate in -short mode")
	}
	specs := map[string]string{
		"minimal":  minimalSpec,
		"petstore": "testdata/petstore.yaml",
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

	t.Run("delegate", func(t *testing.T) {
		dir := t.TempDir()
		code := run([]string{
			"-spec=" + minimalSpec,
			"-out=" + dir,
			"-module=example.com/service",
			"-generate-main",
			"-handler-mode=delegate",
			"-delegate-import=example.com/service/internal/public",
			"-delegate-package=public",
			"-delegate-func=Serve",
		}, &bytes.Buffer{}, os.Stderr)
		require.Equal(t, exitOK, code)

		delegateDir := filepath.Join(dir, "internal", "public")
		require.NoError(t, os.MkdirAll(delegateDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(delegateDir, "public.go"), []byte(`package public

import (
	"context"

	"github.com/cloudwego/hertz/pkg/app"
)

func Serve(ctx context.Context, c *app.RequestContext, operationID string) {}
`), 0o644))

		runGo(t, dir, "mod", "init", "example.com/service")
		runGo(t, dir, "mod", "tidy")
		runGo(t, dir, "build", "./...")
		runGo(t, dir, "vet", "./...")
	})
}

func runGo(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), "go", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "go %v failed:\n%s", args, out)
}
