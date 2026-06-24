package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExported(t *testing.T) {
	cases := map[string]string{
		"createUser":  "CreateUser",
		"getUserById": "GetUserByID",
		"user_id":     "UserID",
		"createdAt":   "CreatedAt",
		"HTTPServer":  "HTTPServer",
		"url":         "URL",
		"list-users":  "ListUsers",
		"@type":       "Type",
		"x-api-key":   "XAPIKey",
		"":            "Generated",
	}
	for in, want := range cases {
		assert.Equal(t, want, Exported(in), "Exported(%q)", in)
	}
}

func TestFileStem(t *testing.T) {
	cases := map[string]string{
		"CreateUserRequest": "create_user_request",
		"User":              "user",
		"ListUsersResponse": "list_users_response",
		"HealthResponse":    "health_response",
	}
	for in, want := range cases {
		assert.Equal(t, want, FileStem(in), "FileStem(%q)", in)
	}
}

func TestParamGoName(t *testing.T) {
	// "type" sanitizes to the exported "Type", which is not a Go keyword, so it
	// is left intact — exported names are inherently keyword-safe.
	assert.Equal(t, "Type", paramGoName("type"))
	assert.Equal(t, "Limit", paramGoName("limit"))
}
