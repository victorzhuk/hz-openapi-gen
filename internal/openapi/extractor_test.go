package openapi

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.wildberries.ru/travel/travel-backend/adv/core/hz-openapi-gen/internal/diag"
)

func TestHertzPath(t *testing.T) {
	assert.Equal(t, "/health", hertzPath("/health"))
	assert.Equal(t, "/users/:id", hertzPath("/users/{id}"))
	assert.Equal(t, "/orgs/:orgId/users/:userId", hertzPath("/orgs/{orgId}/users/{userId}"))
}

func extract(t *testing.T, spec string, strict bool) (SpecModel, *diag.Set) {
	t.Helper()
	doc, err := Load([]byte(spec))
	require.NoError(t, err)
	var d diag.Set
	return Extract(doc, ExtractOptions{Strict: strict}, &d), &d
}

func TestExtractMinimalFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/minimal.yaml")
	require.NoError(t, err)
	spec, d := extract(t, string(data), true)
	require.False(t, d.HasErrors())

	require.Len(t, spec.Operations, 4)
	byFunc := map[string]OpDef{}
	for _, op := range spec.Operations {
		byFunc[op.FuncName] = op
	}

	cu, ok := byFunc["CreateUser"]
	require.True(t, ok)
	assert.Equal(t, "POST", cu.Method)
	assert.Equal(t, "/users", cu.HertzPath)
	assert.Equal(t, "CreateUserRequest", cu.RequestModel)
	assert.Equal(t, "users.go", cu.HandlerFile)

	gid, ok := byFunc["GetUserByID"]
	require.True(t, ok)
	assert.Equal(t, "/users/:id", gid.HertzPath)
	assert.Equal(t, "users.go", gid.HandlerFile)

	gh, ok := byFunc["GetHealth"]
	require.True(t, ok)
	assert.Equal(t, "health.go", gh.HandlerFile)

	require.Len(t, spec.Models, 4)
	var user ModelDef
	for _, m := range spec.Models {
		if m.Name == "User" {
			user = m
		}
	}
	require.Equal(t, "User", user.Name)
	fields := map[string]FieldDef{}
	for _, f := range user.Fields {
		fields[f.JSONName] = f
	}
	assert.Equal(t, "time.Time", fields["createdAt"].GoType)
	assert.True(t, fields["createdAt"].Required)
	assert.True(t, fields["id"].Required)
	assert.False(t, fields["name"].Required)
	assert.True(t, user.UsesTime())
}

func TestSchemaTypeMapping(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: t, version: 1.0.0}
paths: {}
components:
  schemas:
    M:
      type: object
      required: [s]
      properties:
        s: {type: string}
        dt: {type: string, format: date-time}
        by: {type: string, format: byte}
        i32: {type: integer, format: int32}
        i64: {type: integer, format: int64}
        i: {type: integer}
        f: {type: number, format: float}
        d: {type: number}
        bo: {type: boolean}
        arr: {type: array, items: {type: string}}
        mp: {type: object, additionalProperties: {type: integer}}
`
	sm, d := extract(t, spec, true)
	require.False(t, d.HasErrors())
	require.Len(t, sm.Models, 1)

	got := map[string]string{}
	for _, f := range sm.Models[0].Fields {
		got[f.JSONName] = f.GoType
	}
	want := map[string]string{
		"s": "string", "dt": "time.Time", "by": "[]byte",
		"i32": "int32", "i64": "int64", "i": "int",
		"f": "float32", "d": "float64", "bo": "bool",
		"arr": "[]string", "mp": "map[string]int",
	}
	for k, v := range want {
		assert.Equal(t, v, got[k], "field %q", k)
	}
}

func TestExtractPetstoreFixture(t *testing.T) {
	data, err := os.ReadFile("../../testdata/petstore.yaml")
	require.NoError(t, err)
	spec, d := extract(t, string(data), true)
	require.False(t, d.HasErrors())

	require.Len(t, spec.Operations, 4)
	files := map[string]bool{}
	for _, op := range spec.Operations {
		files[op.HandlerFile] = true
	}
	assert.True(t, files["pets.go"])
	assert.True(t, files["store.go"])

	models := map[string]ModelDef{}
	for _, m := range spec.Models {
		models[m.Name] = m
	}
	fieldType := func(model, json string) string {
		for _, f := range models[model].Fields {
			if f.JSONName == json {
				return f.GoType
			}
		}
		return ""
	}
	assert.Equal(t, "int64", fieldType("Pet", "id"))
	assert.Equal(t, "float64", fieldType("Pet", "price"))
	assert.Equal(t, "[]Pet", fieldType("PetList", "items"))
	assert.Equal(t, "map[string]int32", fieldType("Inventory", "counts"))
}

func TestStrictMissingOperationID(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: t, version: 1.0.0}
paths:
  /x:
    get:
      responses: {'200': {description: ok}}
`
	_, d := extract(t, spec, true)
	assert.True(t, d.HasErrors(), "strict mode must reject a missing operationId")

	sm, d2 := extract(t, spec, false)
	assert.False(t, d2.HasErrors())
	require.Len(t, sm.Operations, 1)
	assert.Equal(t, "GetX", sm.Operations[0].FuncName)
	assert.NotEmpty(t, d2.Warnings())
}

func TestDuplicateHandlerName(t *testing.T) {
	spec := `
openapi: 3.0.3
info: {title: t, version: 1.0.0}
paths:
  /a:
    get: {operationId: doThing, responses: {'200': {description: ok}}}
  /b:
    get: {operationId: doThing, responses: {'200': {description: ok}}}
`
	_, d := extract(t, spec, true)
	assert.True(t, d.HasErrors(), "duplicate operationId must be rejected")
}
