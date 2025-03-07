package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type BrunoGenerator struct {
	OutputDir string
	Config    *BrunoCollectionConfig
}

type BrunoMetadata struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Sequence string `json:"seq"`
}

type BrunoRequestData struct {
	URL      string `json:"URL"`
	BodyType string `json:"body"` // focusing on JSON for time being.
	Auth     string `json:"auth"` // currently not supported.
}

type BrunoRequestDocs struct {
	Docs string `json:"docs"`
}

type BrunoCollectionConfig struct {
}

const JSONOutputIndent = "  "

// TODO: consider what configuration may be useful on this.
// Base url / collection level settings.
// We are really generating a collection at a time here.

func NewBrunoGenerator(outputDir string) *BrunoGenerator {
	return &BrunoGenerator{
		OutputDir: outputDir,
		Config:    &BrunoCollectionConfig{},
	}
}

func (g *BrunoGenerator) GenerateRequestFile(route Route) error {

	// TODO: rework this naming paradigm to use the name of the
	fileName := fmt.Sprintf("%s_%s", strings.ToLower(route.Method),
		strings.ReplaceAll(strings.ReplaceAll(route.Path, "/", "_"), ":", "_"))
	filePath := filepath.Join(g.OutputDir, fileName+".bru")

	// Prepare template data
	var bodyJSON string
	if route.RequestBody != nil {
		bodyJSON = generateRequestBodySection(route.RequestBody)
	}

	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}

func generateBrunoMetaDataSection(route *Route) (string, error) {

	metaData := BrunoMetadata{
		Name: route.Name,
		Type: "http",
	}

	jsonBytes, err := json.MarshalIndent(metaData, "", JSONOutputIndent)
	if err != nil {
		return "", err
	}

}

func generateRequestJSONBodySection(requestBody *RequestBody) string {
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
	jsonBytes, err := json.MarshalIndent(body, "", JSONOutputIndent)
	if err != nil {
		return "{}"
	}

	return fmt.Sprintf("json.body %s", string(jsonBytes))

}
