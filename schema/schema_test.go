package schema

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for schema generation
type SimpleStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type StructWithRequired struct {
	Title  string `json:"title" jsonschema:"required,description=The title field"`
	Author string `json:"author" jsonschema:"required"`
	Year   int    `json:"year,omitempty"`
}

type NestedStruct struct {
	ID   string       `json:"id" jsonschema:"required"`
	Data SimpleStruct `json:"data"`
}

type StructWithArray struct {
	Tags []string `json:"tags"`
}

type StructWithMap struct {
	Metadata map[string]string `json:"metadata"`
}

type StructWithPointer struct {
	Optional *string `json:"optional,omitempty"`
}

func TestGenerate(t *testing.T) {
	tests := []struct {
		name       string
		generator  func() (json.RawMessage, error)
		checkProps []string
		checkType  string
	}{
		{
			name:       "simple struct",
			generator:  Generate[SimpleStruct],
			checkProps: []string{"name", "age"},
			checkType:  "object",
		},
		{
			name:       "struct with required fields",
			generator:  Generate[StructWithRequired],
			checkProps: []string{"title", "author", "year"},
			checkType:  "object",
		},
		{
			name:       "nested struct",
			generator:  Generate[NestedStruct],
			checkProps: []string{"id", "data"},
			checkType:  "object",
		},
		{
			name:       "struct with array",
			generator:  Generate[StructWithArray],
			checkProps: []string{"tags"},
			checkType:  "object",
		},
		{
			name:       "struct with map",
			generator:  Generate[StructWithMap],
			checkProps: []string{"metadata"},
			checkType:  "object",
		},
		{
			name:       "struct with pointer",
			generator:  Generate[StructWithPointer],
			checkProps: []string{"optional"},
			checkType:  "object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := tt.generator()
			require.NoError(t, err)
			require.NotEmpty(t, schema)

			// Parse the generated schema
			var parsed map[string]any
			err = json.Unmarshal(schema, &parsed)
			require.NoError(t, err)

			// Check type
			assert.Equal(t, tt.checkType, parsed["type"])

			// Check properties exist
			props, ok := parsed["properties"].(map[string]any)
			require.True(t, ok, "schema should have properties")

			for _, prop := range tt.checkProps {
				assert.Contains(t, props, prop, "schema should contain property %s", prop)
			}
		})
	}
}

func TestGenerate_RequiredFields(t *testing.T) {
	schema, err := Generate[StructWithRequired]()
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(schema, &parsed)
	require.NoError(t, err)

	required, ok := parsed["required"].([]any)
	require.True(t, ok, "schema should have required array")

	// Convert to string slice for easier checking
	requiredStrs := make([]string, len(required))
	for i, r := range required {
		requiredStrs[i] = r.(string)
	}

	assert.Contains(t, requiredStrs, "title")
	assert.Contains(t, requiredStrs, "author")
	assert.NotContains(t, requiredStrs, "year", "year should not be required (omitempty)")
}

func TestGenerate_Description(t *testing.T) {
	schema, err := Generate[StructWithRequired]()
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(schema, &parsed)
	require.NoError(t, err)

	props := parsed["properties"].(map[string]any)
	titleProp := props["title"].(map[string]any)

	assert.Equal(t, "The title field", titleProp["description"])
}

func TestGenerateFromValue(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "from struct value",
			value: &SimpleStruct{},
		},
		{
			name:  "from nested struct value",
			value: &NestedStruct{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := GenerateFromValue(tt.value)
			require.NoError(t, err)
			require.NotEmpty(t, schema)

			var parsed map[string]any
			err = json.Unmarshal(schema, &parsed)
			require.NoError(t, err)

			assert.Equal(t, "object", parsed["type"])
			assert.Contains(t, parsed, "properties")
		})
	}
}

func TestMustGenerate(t *testing.T) {
	t.Run("valid type does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			schema := MustGenerate[SimpleStruct]()
			assert.NotEmpty(t, schema)
		})
	})
}

func TestReflector_DoNotReference(t *testing.T) {
	// Verify that DoNotReference is set, which means nested types
	// are inlined rather than using $ref
	assert.True(t, Reflector.DoNotReference)

	// Generate a schema with nested type
	schema, err := Generate[NestedStruct]()
	require.NoError(t, err)

	// The schema should not contain $ref
	schemaStr := string(schema)
	assert.NotContains(t, schemaStr, "$ref", "schema should not contain $ref when DoNotReference is true")
}

func TestGenerate_ValidJSON(t *testing.T) {
	// Ensure generated schemas are always valid JSON
	types := []func() (json.RawMessage, error){
		Generate[SimpleStruct],
		Generate[StructWithRequired],
		Generate[NestedStruct],
		Generate[StructWithArray],
		Generate[StructWithMap],
		Generate[StructWithPointer],
	}

	for _, gen := range types {
		schema, err := gen()
		require.NoError(t, err)
		assert.True(t, json.Valid(schema), "generated schema should be valid JSON")
	}
}
