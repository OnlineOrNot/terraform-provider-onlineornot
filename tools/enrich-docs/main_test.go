package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestOpenAPIEnumValuesUnmarshalJSON(t *testing.T) {
	t.Parallel()

	var schema Schema
	if err := json.Unmarshal([]byte(`{"enum":["HIGH",true,false,42,null]}`), &schema); err != nil {
		t.Fatalf("unexpected error parsing mixed OpenAPI enum: %v", err)
	}

	expected := OpenAPIEnumValues{"HIGH", "true", "false", "42", "null"}
	if !reflect.DeepEqual(schema.Enum, expected) {
		t.Fatalf("expected enum values %v, got %v", expected, schema.Enum)
	}
}
