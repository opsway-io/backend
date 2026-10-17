package monitors

import (
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
)

func TestExtractSchemaAssertions_FlatObject(t *testing.T) {
	t.Parallel()

	schema := openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewIntegerSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithProperty("description", openapi3.NewStringSchema().WithNullable()).
		WithRequired([]string{"id", "description"})

	assertions := extractSchemaAssertions(schema, "$", 0, 8, make(map[*openapi3.Schema]bool))

	assert.Equal(t, []PreviewOpenAPIAssertion{
		// description is required but nullable, so operator is HAS_KEY
		{Source: "JSON_BODY", Property: "$.description", Operator: "HAS_KEY"},
		// id is required and not nullable, so operator is NOT_NULL
		{Source: "JSON_BODY", Property: "$.id", Operator: "NOT_NULL"},
		// name is optional, so operator is HAS_KEY
		{Source: "JSON_BODY", Property: "$.name", Operator: "HAS_KEY"},
	}, assertions)
}

func TestExtractSchemaAssertions_DeeplyNested(t *testing.T) {
	t.Parallel()

	geoSchema := openapi3.NewObjectSchema().
		WithProperty("lat", openapi3.NewFloat64Schema()).
		WithProperty("lng", openapi3.NewFloat64Schema()).
		WithRequired([]string{"lat", "lng"})

	addressSchema := openapi3.NewObjectSchema().
		WithProperty("city", openapi3.NewStringSchema()).
		WithProperty("geo", geoSchema).
		WithRequired([]string{"city", "geo"})

	userSchema := openapi3.NewObjectSchema().
		WithProperty("address", addressSchema).
		WithRequired([]string{"address"})

	rootSchema := openapi3.NewObjectSchema().
		WithProperty("user", userSchema).
		WithRequired([]string{"user"})

	assertions := extractSchemaAssertions(rootSchema, "$", 0, 8, make(map[*openapi3.Schema]bool))

	assert.Equal(t, []PreviewOpenAPIAssertion{
		{Source: "JSON_BODY", Property: "$.user", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.user.address", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.user.address.city", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.user.address.geo", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.user.address.geo.lat", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.user.address.geo.lng", Operator: "NOT_NULL"},
	}, assertions)
}

func TestExtractSchemaAssertions_ArrayRoot(t *testing.T) {
	t.Parallel()

	itemSchema := openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewIntegerSchema()).
		WithProperty("name", openapi3.NewStringSchema()).
		WithRequired([]string{"id"})

	rootSchema := openapi3.NewArraySchema().WithItems(itemSchema)

	assertions := extractSchemaAssertions(rootSchema, "$", 0, 8, make(map[*openapi3.Schema]bool))

	assert.Equal(t, []PreviewOpenAPIAssertion{
		{Source: "JSON_BODY", Property: "$", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$[0].id", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$[0].name", Operator: "HAS_KEY"},
	}, assertions)
}

func TestExtractSchemaAssertions_NestedArray(t *testing.T) {
	t.Parallel()

	tagSchema := openapi3.NewObjectSchema().
		WithProperty("tagId", openapi3.NewIntegerSchema()).
		WithRequired([]string{"tagId"})

	rootSchema := openapi3.NewObjectSchema().
		WithProperty("tags", openapi3.NewArraySchema().WithItems(tagSchema))

	assertions := extractSchemaAssertions(rootSchema, "$", 0, 8, make(map[*openapi3.Schema]bool))

	assert.Equal(t, []PreviewOpenAPIAssertion{
		{Source: "JSON_BODY", Property: "$.tags", Operator: "HAS_KEY"},
		{Source: "JSON_BODY", Property: "$.tags[0].tagId", Operator: "NOT_NULL"},
	}, assertions)
}

func TestExtractSchemaAssertions_AllOf(t *testing.T) {
	t.Parallel()

	baseSchema := openapi3.NewObjectSchema().
		WithProperty("id", openapi3.NewIntegerSchema()).
		WithRequired([]string{"id"})

	childSchema := openapi3.NewObjectSchema().
		WithProperty("childName", openapi3.NewStringSchema())

	composedSchema := openapi3.NewAllOfSchema(baseSchema, childSchema)

	assertions := extractSchemaAssertions(composedSchema, "$", 0, 8, make(map[*openapi3.Schema]bool))

	assert.Equal(t, []PreviewOpenAPIAssertion{
		{Source: "JSON_BODY", Property: "$.id", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.childName", Operator: "HAS_KEY"},
	}, assertions)
}

func TestExtractSchemaAssertions_CircularReference(t *testing.T) {
	t.Parallel()

	nodeSchema := openapi3.NewObjectSchema().
		WithProperty("name", openapi3.NewStringSchema())

	// Create cycle: node has child of type node
	nodeSchema.WithProperty("child", nodeSchema)

	// Should not enter an infinite loop
	assertions := extractSchemaAssertions(nodeSchema, "$", 0, 8, make(map[*openapi3.Schema]bool))

	assert.NotEmpty(t, assertions)
	assert.Equal(t, "$.child", assertions[0].Property)
	assert.Equal(t, "$.name", assertions[1].Property)
}

func TestIsJSONMediaType(t *testing.T) {
	t.Parallel()

	assert.True(t, isJSONMediaType("application/json"))
	assert.True(t, isJSONMediaType("application/json; charset=utf-8"))
	assert.True(t, isJSONMediaType("application/problem+json"))
	assert.True(t, isJSONMediaType("application/vnd.api+json"))
	assert.False(t, isJSONMediaType("text/html"))
	assert.False(t, isJSONMediaType("application/xml"))
	assert.False(t, isJSONMediaType("multipart/form-data"))
}

func TestDeduplicateAssertions(t *testing.T) {
	t.Parallel()

	input := []PreviewOpenAPIAssertion{
		{Source: "JSON_BODY", Property: "$.id", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.id", Operator: "NOT_NULL"},
		{Source: "JSON_BODY", Property: "$.name", Operator: "HAS_KEY"},
	}

	deduped := deduplicateAssertions(input)
	assert.Len(t, deduped, 2)
	assert.Equal(t, "$.id", deduped[0].Property)
	assert.Equal(t, "$.name", deduped[1].Property)
}
