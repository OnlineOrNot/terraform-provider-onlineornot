// enrich-docs parses the OpenAPI spec and enriches Terraform documentation
// with enum values for fields that have them.
//
// Strategy: For each resource type (check, heartbeat, etc.), we extract the
// enum values from the POST endpoint's request body schema, which gives us
// the correct context-aware enum values for that resource.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	openAPIURL = "https://raw.githubusercontent.com/OnlineOrNot/api-schemas/main/openapi.json"
	docsDir    = "docs"
)

// OpenAPI spec structures (minimal for our needs)
type OpenAPISpec struct {
	Paths      map[string]PathItem `json:"paths"`
	Components Components          `json:"components"`
}

type Components struct {
	Schemas map[string]Schema `json:"schemas"`
}

type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

type Operation struct {
	RequestBody *RequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*Response `json:"responses,omitempty"`
}

type RequestBody struct {
	Content map[string]MediaType `json:"content,omitempty"`
}

type Response struct {
	Content map[string]MediaType `json:"content,omitempty"`
}

type MediaType struct {
	Schema Schema `json:"schema,omitempty"`
}

// SchemaType handles OpenAPI type which can be string or array of strings
type SchemaType struct {
	Single   string
	Multiple []string
}

func (st *SchemaType) UnmarshalJSON(data []byte) error {
	// Try string first
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		st.Single = s
		return nil
	}

	// Try array of strings
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		st.Multiple = arr
		return nil
	}

	return nil // Ignore if neither works
}

type Schema struct {
	Type        SchemaType        `json:"type,omitempty"`
	Enum        []string          `json:"enum,omitempty"`
	Description string            `json:"description,omitempty"`
	Properties  map[string]Schema `json:"properties,omitempty"`
	Items       *Schema           `json:"items,omitempty"`
	Ref         string            `json:"$ref,omitempty"`
	AllOf       []Schema          `json:"allOf,omitempty"`
	OneOf       []Schema          `json:"oneOf,omitempty"`
	AnyOf       []Schema          `json:"anyOf,omitempty"`
}

// EnumInfo holds enum values for a field
type EnumInfo struct {
	Values []string
}

// ResourceEnums maps resource name -> field name -> enum info
// Field names can be qualified like "assertions.type" for nested fields
type ResourceEnums map[string]map[string]EnumInfo

// Mapping from Terraform resource names to OpenAPI paths
var resourcePaths = map[string]string{
	"check":                             "/v1/checks",
	"heartbeat":                         "/v1/heartbeats",
	"maintenance_window":                "/v1/maintenance-windows",
	"webhook":                           "/v1/webhooks",
	"status_page":                       "/v1/status_pages",
	"status_page_component":             "/v1/status_pages/{status_page_id}/components",
	"status_page_component_group":       "/v1/status_pages/{status_page_id}/groups",
	"status_page_incident":              "/v1/status_pages/{status_page_id}/incidents",
	"status_page_scheduled_maintenance": "/v1/status_pages/{status_page_id}/scheduled_maintenance",
}

// Mapping from Terraform data source names to OpenAPI paths
var dataSourcePaths = map[string]string{
	"checks":              "/v1/checks",
	"heartbeats":          "/v1/heartbeats",
	"maintenance_windows": "/v1/maintenance-windows",
	"webhooks":            "/v1/webhooks",
	"status_pages":        "/v1/status_pages",
	"user":                "/v1/users/{user_id}",
	"users":               "/v1/users",
}

func main() {
	fmt.Println("Fetching OpenAPI spec...")
	spec, err := fetchOpenAPISpec()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching OpenAPI spec: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Extracting enum fields per resource...")
	resourceEnums := extractResourceEnums(spec)

	for resource, enums := range resourceEnums {
		if len(enums) > 0 {
			fmt.Printf("  %s: %d enum fields\n", resource, len(enums))
			for field, info := range enums {
				fmt.Printf("    %s: %v\n", field, info.Values)
			}
		}
	}

	fmt.Println("\nEnriching documentation...")
	err = enrichDocs(resourceEnums, spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error enriching docs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}

func fetchOpenAPISpec() (*OpenAPISpec, error) {
	resp, err := http.Get(openAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	var spec OpenAPISpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return &spec, nil
}

func extractResourceEnums(spec *OpenAPISpec) ResourceEnums {
	result := make(ResourceEnums)

	// Extract for resources (from POST request body)
	for resource, path := range resourcePaths {
		enums := extractEnumsFromPath(spec, path, "POST")
		if len(enums) > 0 {
			result[resource] = enums
		}
	}

	// Extract for data sources (from GET response body)
	for dataSource, path := range dataSourcePaths {
		enums := extractEnumsFromPath(spec, path, "GET")
		if len(enums) > 0 {
			result[dataSource] = enums
		}
	}

	return result
}

func extractEnumsFromPath(spec *OpenAPISpec, path string, method string) map[string]EnumInfo {
	enums := make(map[string]EnumInfo)

	pathItem, ok := spec.Paths[path]
	if !ok {
		return enums
	}

	var op *Operation
	switch method {
	case "GET":
		op = pathItem.Get
	case "POST":
		op = pathItem.Post
	case "PATCH":
		op = pathItem.Patch
	}

	if op == nil {
		return enums
	}

	// For POST/PATCH, extract from request body
	if method == "POST" || method == "PATCH" {
		if op.RequestBody != nil {
			for _, mediaType := range op.RequestBody.Content {
				extractEnumsFromSchemaFlat(&mediaType.Schema, enums, spec, "")
			}
		}
	}

	// For GET, extract from response body
	if method == "GET" {
		if response, ok := op.Responses["200"]; ok && response != nil {
			for _, mediaType := range response.Content {
				extractEnumsFromSchemaFlat(&mediaType.Schema, enums, spec, "")
			}
		}
	}

	// Post-process: strip "result." prefix from enum keys (API wrapper)
	// and "result_info." prefix (pagination wrapper)
	cleaned := make(map[string]EnumInfo)
	for key, value := range enums {
		cleanKey := strings.TrimPrefix(key, "result.")
		cleanKey = strings.TrimPrefix(cleanKey, "result_info.")
		cleaned[cleanKey] = value
	}

	// For data sources that return arrays, fields might be nested under
	// a plural name (e.g., "checks.status"). Add mappings without the
	// plural prefix as well for cases where the nested context matches.
	// Also add with common nested schema names.
	for key, value := range enums {
		cleanKey := strings.TrimPrefix(key, "result.")
		cleanKey = strings.TrimPrefix(cleanKey, "result_info.")

		// Add both the clean key and potential nested variants
		cleaned[cleanKey] = value

		// For list responses, the items are often accessed via a nested schema
		// named after the resource (singular or plural)
		// Add variants that match Terraform's nested schema naming
	}

	return cleaned
}

func extractEnumsFromSchemaFlat(schema *Schema, enums map[string]EnumInfo, spec *OpenAPISpec, prefix string) {
	if schema == nil {
		return
	}

	// Handle $ref
	if schema.Ref != "" {
		refName := strings.TrimPrefix(schema.Ref, "#/components/schemas/")
		if refSchema, ok := spec.Components.Schemas[refName]; ok {
			extractEnumsFromSchemaFlat(&refSchema, enums, spec, prefix)
		}
		return
	}

	// Handle allOf, oneOf, anyOf
	for i := range schema.AllOf {
		extractEnumsFromSchemaFlat(&schema.AllOf[i], enums, spec, prefix)
	}
	for i := range schema.OneOf {
		extractEnumsFromSchemaFlat(&schema.OneOf[i], enums, spec, prefix)
	}
	for i := range schema.AnyOf {
		extractEnumsFromSchemaFlat(&schema.AnyOf[i], enums, spec, prefix)
	}

	// Process properties at this level
	for propName, propSchema := range schema.Properties {
		// Build the qualified name for nested fields
		qualifiedName := propName
		if prefix != "" {
			qualifiedName = prefix + "." + propName
		}

		// Record enum if present
		if len(propSchema.Enum) > 0 {
			enums[qualifiedName] = EnumInfo{Values: propSchema.Enum}
		}

		// Recurse into nested objects (like assertions)
		if propSchema.Type.Single == "object" || len(propSchema.Properties) > 0 {
			extractEnumsFromSchemaFlat(&propSchema, enums, spec, propName)
		}

		// Handle array items - use the property name as prefix for items
		if propSchema.Items != nil {
			extractEnumsFromSchemaFlat(propSchema.Items, enums, spec, propName)
		}
	}

	// Handle array items at top level
	if schema.Items != nil {
		extractEnumsFromSchemaFlat(schema.Items, enums, spec, prefix)
	}
}

func enrichDocs(resourceEnums ResourceEnums, spec *OpenAPISpec) error {
	// Process resources
	resourcesDir := filepath.Join(docsDir, "resources")
	entries, err := os.ReadDir(resourcesDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read resources dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		resourceName := strings.TrimSuffix(entry.Name(), ".md")
		enums := resourceEnums[resourceName]
		if enums == nil {
			enums = make(map[string]EnumInfo)
		}

		filePath := filepath.Join(resourcesDir, entry.Name())
		if err := enrichMarkdownFile(filePath, enums); err != nil {
			return fmt.Errorf("failed to process %s: %w", filePath, err)
		}
	}

	// Process data sources
	dataSourcesDir := filepath.Join(docsDir, "data-sources")
	entries, err = os.ReadDir(dataSourcesDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read data-sources dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		dataSourceName := strings.TrimSuffix(entry.Name(), ".md")
		enums := resourceEnums[dataSourceName]
		if enums == nil {
			enums = make(map[string]EnumInfo)
		}

		filePath := filepath.Join(dataSourcesDir, entry.Name())
		if err := enrichMarkdownFile(filePath, enums); err != nil {
			return fmt.Errorf("failed to process %s: %w", filePath, err)
		}
	}

	return nil
}

func enrichMarkdownFile(filePath string, enums map[string]EnumInfo) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	original := string(content)
	enriched := enrichMarkdownContent(original, enums)

	if enriched != original {
		fmt.Printf("  Updated: %s\n", filePath)
		return os.WriteFile(filePath, []byte(enriched), 0644)
	}

	return nil
}

func enrichMarkdownContent(content string, enums map[string]EnumInfo) string {
	lines := strings.Split(content, "\n")
	result := make([]string, 0, len(lines))

	// Track the current nested context (e.g., "assertions" when inside ### Nested Schema for `assertions`)
	currentContext := ""
	nestedHeaderPattern := regexp.MustCompile("^### Nested Schema for `([a-z_]+)`")
	mainHeaderPattern := regexp.MustCompile("^## Schema")

	// Match Terraform doc attribute lines like:
	// - `field_name` (String) Description
	attrPattern := regexp.MustCompile("^(- `([a-z_]+)` \\([^)]+\\))(.*)$")

	for _, line := range lines {
		// Check for main schema header - reset context
		if mainHeaderPattern.MatchString(line) {
			currentContext = ""
		}

		// Check for nested schema headers
		if matches := nestedHeaderPattern.FindStringSubmatch(line); len(matches) > 1 {
			currentContext = matches[1]
		}

		// Try to match attribute lines
		if matches := attrPattern.FindStringSubmatch(line); len(matches) >= 4 {
			prefix := matches[1]    // - `field_name` (Type)
			fieldName := matches[2] // field_name
			rest := matches[3]      // Description or empty

			// Build qualified field name based on context
			lookupName := fieldName
			if currentContext != "" {
				lookupName = currentContext + "." + fieldName
			}

			// Check if this field has enum values
			// Try qualified name first, then fall back to unqualified
			enumInfo, ok := enums[lookupName]
			if !ok && currentContext != "" {
				// Try without context (for data sources where result.X becomes just X)
				enumInfo, ok = enums[fieldName]
			}
			if ok && !strings.Contains(rest, "Must be one of:") {
				// Format enum values
				quotedValues := make([]string, len(enumInfo.Values))
				for i, v := range enumInfo.Values {
					quotedValues[i] = fmt.Sprintf("`%s`", v)
				}

				// Sort for consistent output
				sort.Strings(quotedValues)
				enumSuffix := fmt.Sprintf(" Must be one of: %s.", strings.Join(quotedValues, ", "))

				// Append enum values to description
				rest = strings.TrimSpace(rest)
				if rest == "" {
					line = prefix + enumSuffix
				} else {
					// Remove trailing period if present, then add our suffix
					rest = strings.TrimSuffix(rest, ".")
					line = prefix + " " + rest + "." + enumSuffix
				}
			}
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
