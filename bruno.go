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
	Method   string `json:"method"`
}

type BrunoRequestDocs struct {
	Docs string `json:"docs"`
}

type BrunoCollectionConfig struct {
	BaseURL string
}

const JSONOutputIndent = "  "

// NewBrunoGenerator creates a new Bruno generator instance
func NewBrunoGenerator(outputDir string, baseURL string) *BrunoGenerator {
	return &BrunoGenerator{
		OutputDir: outputDir,
		Config: &BrunoCollectionConfig{
			BaseURL: baseURL,
		},
	}
}

// GenerateRequestFile generates a Bruno request file for a given route
func (g *BrunoGenerator) GenerateRequestFile(route *Route) error {

	// TODO: rework this naming paradigm to use the name of the route
	fileName := fmt.Sprintf("%s_%s", strings.ToLower(route.Method),
		strings.ReplaceAll(strings.ReplaceAll(route.Path, "/", "_"), ":", "_"))
	filePath := filepath.Join(g.OutputDir, fileName+".bru")

	metaDataSectionString, err := g.generateBrunoMetaDataSection(route)
	if err != nil {
		return err
	}

	requestSectionString, err := g.generateBrunoRequestSection(route)
	if err != nil {
		return err
	}

	var bodyJSONString string
	if route.RequestBody != nil {
		bodyJSONString, err = g.generateRequestJSONBodySection(route.RequestBody)
		if err != nil {
			return err
		}
	}

	docsSectionString, err := g.GenerateDocsSection(route)
	if err != nil {
		return err
	}

	// Make sure the output directory exists.
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	// Generate the unique file path.
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	sections := []string{
		metaDataSectionString,
		requestSectionString,
		bodyJSONString,
		docsSectionString,
	}

	content := strings.Join(sections, "\n\n")

	_, err = file.WriteString(content)
	return err
}

// generateBrunoMetaDataSection creates the metadata section for a Bruno request file
func (g *BrunoGenerator) generateBrunoMetaDataSection(route *Route) (string, error) {
	meta := BrunoMetadata{
		Name: route.Name,
		Type: "http",
	}
	jsonBytes, err := json.MarshalIndent(meta, "", JSONOutputIndent)
	if err != nil {
		return "", err
	}
	jsonString := strings.ReplaceAll(string(jsonBytes), `"`, "")
	return fmt.Sprintf("meta %s", jsonString), nil
}

// generateBrunoRequestSection creates the request section for a Bruno request file
func (g *BrunoGenerator) generateBrunoRequestSection(route *Route) (string, error) {
	requestData := BrunoRequestData{
		URL:  g.Config.BaseURL + route.Path,
		Auth: "none",
	}

	if route.RequestBody != nil {
		requestData.BodyType = "json"
	}

	jsonBytes, err := json.MarshalIndent(requestData, "", JSONOutputIndent)
	if err != nil {
		return "", err
	}

	jsonString := strings.ReplaceAll(string(jsonBytes), `"`, "")
	methodPrefix := strings.ToLower(route.Method)

	return fmt.Sprintf("%s %s", methodPrefix, jsonString), nil
}

// generateRequestJSONBodySection creates the JSON body section for a Bruno request file
func (g *BrunoGenerator) generateRequestJSONBodySection(requestBody *RequestBody) (string, error) {
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
		return "", err
	}

	return fmt.Sprintf("json.body %s", string(jsonBytes)), nil
}

// GenerateDocsSection generates documentation section for a Bruno request file
func (g *BrunoGenerator) GenerateDocsSection(route *Route) (string, error) {
	docs := BrunoRequestDocs{
		Docs: route.Description,
	}

	jsonBytes, err := json.MarshalIndent(docs, "", JSONOutputIndent)
	if err != nil {
		return "", err
	}

	jsonString := strings.ReplaceAll(string(jsonBytes), `"`, "")
	return fmt.Sprintf("docs %s", jsonString), nil
}

// GenerateCollection generates a complete Bruno collection
func (g *BrunoGenerator) GenerateCollection(routes []*Route) error {
	// Create collection directory if it doesn't exist
	if err := os.MkdirAll(g.OutputDir, 0755); err != nil {
		return err
	}

	// Generate each request file
	for _, route := range routes {
		if err := g.GenerateRequestFile(route); err != nil {
			return err
		}
	}

	return nil
}
