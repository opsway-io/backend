package openapi

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const v3Spec = `{
  "openapi": "3.0.0",
  "servers": [{"url": "/api/v1"}],
  "paths": {
    "/users/{id}": {
      "get": {
        "summary": "Get user",
        "responses": {
          "404": {"description": "nope"},
          "200": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/User"}}}}
        }
      }
    },
    "/users": {
      "post": {
        "operationId": "createUser",
        "requestBody": {"content": {"application/json": {"schema": {"$ref": "#/components/schemas/User"}}}},
        "responses": {"201": {"description": "created"}}
      }
    }
  },
  "components": {
    "schemas": {
      "User": {
        "type": "object",
        "required": ["id", "email"],
        "properties": {
          "id": {"type": "integer"},
          "email": {"type": "string", "format": "email"},
          "tags": {"type": "array", "items": {"type": "string"}}
        }
      }
    },
    "securitySchemes": {
      "oauth": {"type": "oauth2", "flows": {"clientCredentials": {"tokenUrl": "https://auth.example.com/token"}}}
    }
  }
}`

func TestParseOpenAPI3(t *testing.T) {
	specURL, _ := url.Parse("https://api.example.com/docs/openapi.json")

	p, err := Parse([]byte(v3Spec), specURL)
	require.NoError(t, err)

	require.Len(t, p.Endpoints, 2)
	assert.Equal(t, "https://api.example.com/api/v1", p.BaseURL)

	post := p.Endpoints[0]
	assert.Equal(t, "POST", post.Method)
	assert.Equal(t, "/api/v1/users", post.Path)
	assert.Equal(t, "createUser", post.Summary)
	assert.Equal(t, "201", post.StatusCode)
	require.NotNil(t, post.RequestBody)
	assert.JSONEq(t, `{"id":0,"email":"user@example.com","tags":["string"]}`, *post.RequestBody)

	get := p.Endpoints[1]
	assert.Equal(t, "GET", get.Method)
	assert.Equal(t, "/api/v1/users/{id}", get.Path)
	assert.Equal(t, "200", get.StatusCode)
	assert.Nil(t, get.RequestBody)
	assert.Equal(t, []Assertion{
		{Source: "STATUS_CODE", Operator: "EQUAL", Target: "200"},
		{Source: "JSON_BODY", Property: "$.id", Operator: "HAS_KEY"},
		{Source: "JSON_BODY", Property: "$.email", Operator: "HAS_KEY"},
	}, get.Assertions)

	require.NotNil(t, p.Auth)
	assert.Equal(t, "OAUTH2_CLIENT_CREDENTIALS", p.Auth.Method)
	assert.Equal(t, "https://auth.example.com/token", p.Auth.TokenURL)
}

const v2Spec = `
swagger: "2.0"
host: petstore.example.com
basePath: /v2
schemes: [https]
securityDefinitions:
  basicAuth:
    type: basic
paths:
  /pet:
    put:
      summary: Update pet
      parameters:
        - in: body
          name: body
          schema:
            type: object
            properties:
              name: {type: string, example: doggie}
      responses:
        "200": {description: ok}
`

func TestParseSwagger2YAML(t *testing.T) {
	p, err := Parse([]byte(v2Spec), nil)
	require.NoError(t, err)

	require.Len(t, p.Endpoints, 1)
	e := p.Endpoints[0]
	assert.Equal(t, "PUT", e.Method)
	assert.Equal(t, "/v2/pet", e.Path)
	assert.Equal(t, "Update pet", e.Summary)
	require.NotNil(t, e.RequestBody)
	assert.JSONEq(t, `{"name":"doggie"}`, *e.RequestBody)
	assert.Equal(t, "https://petstore.example.com/v2", p.BaseURL)

	require.NotNil(t, p.Auth)
	assert.Equal(t, "BASIC", p.Auth.Method)
}

func TestParseInvalid(t *testing.T) {
	_, err := Parse([]byte(`{"hello":"world"}`), nil)
	assert.ErrorIs(t, err, ErrInvalidSpec)

	_, err = Parse([]byte(`not: [valid`), nil)
	assert.ErrorIs(t, err, ErrInvalidSpec)
}
