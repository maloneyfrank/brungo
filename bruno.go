package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// BrunoGenerator generates Bruno .bru files from route information
type BrunoGenerator struct {
	OutputDir string
}

// NewBrunoGenerator creates a new BrunoGenerator
func NewBrunoGenerator(outputDir string) *BrunoGenerator {
	return &BrunoGenerator{
		OutputDir: outputDir,
	}
}

// BruFileTemplate defines the template for .bru files
const BruFileTemplate = `
meta {
  name: {{.Name}}
  type: http
  seq: 1
}

{{.Method | toLower}} {
  url: {{.URL}}
  body: json
  auth: none
}
{{if .HasBody}}
body:json {
  {{.Body}}
}
{{end}}
{{if .Description}}
docs {
  {{.Description}}
}
{{end}}
`

// GenerateRequestFile generates a Bruno .bru file for a single route
func (g *BrunoGenerator) GenerateRequestFile(route Route) error {
	// Create a unique name for the file
	fileName := fmt.Sprintf("%s_%s", strings.ToLower(route.Method),
		strings.ReplaceAll(strings.ReplaceAll(route.Path, "/", "_"), ":", "_"))
	filePath := filepath.Join(g.OutputDir, fileName+".bru")

	// Prepare template data
	var bodyJSON string
	if route.RequestBody != nil {
		bodyJSON = generateRequestBody(route.RequestBody)
	}

	data := struct {
		Name        string
		Method      string
		URL         string
		HasBody     bool
		Body        string
		Description string
	}{
		Name:        fmt.Sprintf("%s %s", route.Method, route.Path),
		Method:      route.Method,
		URL:         "{{baseUrl}}" + route.Path,
		HasBody:     route.RequestBody != nil,
		Body:        bodyJSON,
		Description: route.Description,
	}

	// Create template with custom function
	tmpl := template.New("brufile")
	tmpl.Funcs(template.FuncMap{
		"toLower": strings.ToLower,
	})

	t, err := tmpl.Parse(BruFileTemplate)
	if err != nil {
		return fmt.Errorf("error parsing template: %v", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Execute the template
	if err := t.Execute(file, data); err != nil {
		return err
	}

	return nil
}

// generateRequestBody creates a JSON template from struct fields
func generateRequestBody(requestBody *RequestBody) string {
	body := make(map[string]interface{})

	for _, field := range requestBody.Fields {
		var defaultValue interface{}

		// Generate default values based on field type
		switch strings.ToLower(field.Type) {
		case "string":
			defaultValue = ""
		case "int", "int64", "int32", "float64", "float32":
			defaultValue = 0
		case "bool":
			defaultValue = false
		case "array", "slice":
			defaultValue = []interface{}{}
		case "map":
			defaultValue = map[string]interface{}{}
		default:
			defaultValue = nil
		}

		body[field.JSONName] = defaultValue
	}

	// Convert to JSON
	jsonBytes, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(jsonBytes)
}
