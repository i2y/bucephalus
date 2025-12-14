// Package schema provides JSON Schema generation from Go types.
package schema

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// Reflector is configured for LLM tool/response schemas.
// DoNotReference inlines all definitions to avoid $ref.
var Reflector = &jsonschema.Reflector{
	DoNotReference: true,
}

// Generate creates a JSON Schema from a Go type.
// The type should be a struct with json and jsonschema tags.
//
// Example:
//
//	type Book struct {
//	    Title  string `json:"title" jsonschema:"required,description=The book title"`
//	    Author string `json:"author" jsonschema:"required"`
//	    Year   int    `json:"year,omitempty"`
//	}
//
//	schema, err := schema.Generate[Book]()
func Generate[T any]() (json.RawMessage, error) {
	var zero T
	schema := Reflector.Reflect(&zero)
	return json.Marshal(schema)
}

// GenerateFromValue creates a JSON Schema from a value.
// This is useful when you have a value instead of a type.
func GenerateFromValue(v any) (json.RawMessage, error) {
	schema := Reflector.Reflect(v)
	return json.Marshal(schema)
}

// MustGenerate is like Generate but panics on error.
// Useful for package-level schema definitions.
func MustGenerate[T any]() json.RawMessage {
	schema, err := Generate[T]()
	if err != nil {
		panic(err)
	}
	return schema
}
