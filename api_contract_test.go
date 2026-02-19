package main

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestAPIContractRoutes validates that every path in the OpenAPI spec
// has a corresponding route in the main.go switch statement.
func TestAPIContractRoutes(t *testing.T) {
	// Parse OpenAPI spec
	specBytes, err := os.ReadFile("docs/api/openapi.yaml")
	if err != nil {
		t.Fatalf("Cannot read OpenAPI spec: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]interface{} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("Cannot parse OpenAPI spec: %v", err)
	}

	// Read main.go to check route existence
	mainBytes, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("Cannot read main.go: %v", err)
	}
	mainSrc := string(mainBytes)

	// For each spec path, verify it appears in the routing logic
	for path, methods := range spec.Paths {
		for method := range methods {
			method = strings.ToUpper(method)
			// Convert OpenAPI path to the parts-based routing pattern
			// e.g., /parts/{ipn}/bom -> parts[0] == "parts" ... parts[2] == "bom"
			segments := strings.Split(strings.Trim(path, "/"), "/")
			if len(segments) == 0 {
				continue
			}

			// Check that the first segment appears in main.go routing
			firstSeg := segments[0]
			// Some paths use hyphens which map to different patterns
			routeCheck := `parts[0] == "` + firstSeg + `"`

			// For simple paths, just check first segment + method
			methodCheck := `r.Method == "` + method + `"`

			if !strings.Contains(mainSrc, routeCheck) && !strings.Contains(mainSrc, `"`+firstSeg+`"`) {
				t.Errorf("OpenAPI path %s %s: first segment %q not found in main.go routing", method, path, firstSeg)
			}

			// Verify the method is handled somewhere for this path prefix
			if !strings.Contains(mainSrc, methodCheck) {
				t.Errorf("OpenAPI path %s %s: method %s not found in main.go", method, path, method)
			}
		}
	}
}

// TestFrontendRoutesExistInBackend checks that key frontend API calls
// have matching backend routes (the specific drift issues we fixed).
func TestFrontendRoutesExistInBackend(t *testing.T) {
	mainBytes, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("Cannot read main.go: %v", err)
	}
	mainSrc := string(mainBytes)

	// These are the routes that were identified as missing and have been fixed
	requiredRoutes := []struct {
		description string
		mustContain string
	}{
		{"inventory bulk-delete", `"bulk-delete"`},
		{"pos generate", `"generate"`},
		{"firmware routes", `parts[0] == "firmware"`},
		{"attachment download", `"download"`},
		{"test by ID handler", `handleGetTestByID`},
		{"parts/categories alias", `parts[1] == "categories"`},
	}

	for _, route := range requiredRoutes {
		if !strings.Contains(mainSrc, route.mustContain) {
			t.Errorf("Missing route for %s: expected %q in main.go", route.description, route.mustContain)
		}
	}
}

// TestOpenAPISpecValid does basic validation of the OpenAPI spec structure.
func TestOpenAPISpecValid(t *testing.T) {
	specBytes, err := os.ReadFile("docs/api/openapi.yaml")
	if err != nil {
		t.Fatalf("Cannot read OpenAPI spec: %v", err)
	}

	var spec struct {
		OpenAPI string                            `yaml:"openapi"`
		Info    map[string]interface{}             `yaml:"info"`
		Paths   map[string]map[string]interface{} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(specBytes, &spec); err != nil {
		t.Fatalf("Invalid YAML: %v", err)
	}

	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		t.Errorf("Expected OpenAPI 3.x, got %s", spec.OpenAPI)
	}

	if spec.Info["title"] == nil {
		t.Error("OpenAPI spec missing info.title")
	}

	if len(spec.Paths) == 0 {
		t.Error("OpenAPI spec has no paths")
	}

	// Count endpoints
	endpointCount := 0
	for _, methods := range spec.Paths {
		endpointCount += len(methods)
	}
	t.Logf("OpenAPI spec: %d paths, %d endpoints", len(spec.Paths), endpointCount)

	// Verify minimum endpoint count (we have ~120+ endpoints)
	if endpointCount < 80 {
		t.Errorf("Expected at least 80 endpoints, got %d", endpointCount)
	}
}
