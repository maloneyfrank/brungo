package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
)

var (
	namePattern        = regexp.MustCompile(`@name\s+(.+)`)
	routePattern       = regexp.MustCompile(`@route\s+([A-Z]+)\s+(.+)`)
	descriptionPattern = regexp.MustCompile(`@description\s+(.+)`)
	bodyPattern        = regexp.MustCompile(`@body\s+(\w+)`)
)

// Parser extracts information about API routes
type Parser struct {
	routes []*Route
}

// NewParser creates a new Parser
func NewParser() *Parser {
	return &Parser{
		routes: []*Route{},
	}
}

// ParseDirectory parses all Go files in a directory
func (p *Parser) ParseDirectory(dirPath string) ([]*Route, error) {
	// First, find all handler functions and their annotations to create route stubs
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			if err := p.FindHandlers(path); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Then, go through all files again to find struct definitions referenced by the routes
	for i, route := range p.routes {
		// Skip routes that don't need a request body
		if route.BodyType == "" {
			continue
		}

		// Look for the struct in all files
		requestBody, err := p.FindStruct(dirPath, route.BodyType)
		if err != nil {
			return nil, err
		}

		if requestBody != nil {
			p.routes[i].RequestBody = requestBody
		}
	}

	return p.routes, nil
}

// FindHandlers parses a file to find handler functions and their annotations
func (p *Parser) FindHandlers(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// TODO: right now this lets us annotate any function, not just handler funcs.

	// Extract handler annotations
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for function declarations (handlers)
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			// Skip if no comments
			if funcDecl.Doc == nil {
				return true
			}

			handlerName := funcDecl.Name.Name

			// Extract annotations from comments
			annotations := p.extractAnnotations(funcDecl.Doc)

			// Check if we have route information
			method, hasMethod := annotations["route_method"]
			path, hasPath := annotations["route_path"]

			// Only process functions with a @route annotation
			if hasMethod && hasPath {
				route := &Route{
					Name:        annotations["name"],
					Method:      method,
					Path:        path,
					Handler:     handlerName,
					Description: annotations["description"],
					BodyType:    annotations["body"], // Store the body type name to be resolved later
					Tags:        make(map[string]string),
				}

				p.routes = append(p.routes, route)
				fmt.Printf("Found route: %s %s in handler %s\n", method, path, handlerName)
			}
		}
		return true
	})

	return nil
}

// FindStruct searches for a specific struct definition across all files
func (p *Parser) FindStruct(dirPath, structName string) (*RequestBody, error) {
	var foundStruct *RequestBody

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if already found or not a Go file
		if foundStruct != nil || info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Try to find the struct in this file
		requestBody, err := p.ParseStructFromFile(path, structName)
		if err != nil {
			return err
		}

		if requestBody != nil {
			foundStruct = requestBody
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return foundStruct, nil
}

// ParseStructFromFile parses a file looking for a specific struct
func (p *Parser) ParseStructFromFile(filePath, structName string) (*RequestBody, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var requestBody *RequestBody

	// Look for the specific struct
	ast.Inspect(node, func(n ast.Node) bool {
		// Once found, we can stop inspecting
		if requestBody != nil {
			return false
		}

		// Look for struct definitions
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			// Only process if this is the struct we're looking for
			if typeSpec.Name.Name != structName {
				return true
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true // Not a struct
			}

			// Extract struct fields
			fields := []RequestBodyField{}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					continue // Skip embedded fields
				}

				fieldName := field.Names[0].Name

				// Get field type as string
				var fieldType string
				switch t := field.Type.(type) {
				case *ast.Ident:
					fieldType = t.Name
				case *ast.SelectorExpr:
					fieldType = t.Sel.Name
				case *ast.ArrayType:
					fieldType = "array"
				case *ast.MapType:
					fieldType = "map"
				default:
					fieldType = "unknown"
				}

				// Parse struct tags
				tags := make(map[string]string)
				jsonName := fieldName
				required := false

				if field.Tag != nil && len(field.Tag.Value) > 0 {
					tagValue := strings.Trim(field.Tag.Value, "`")
					structTags := reflect.StructTag(tagValue)

					// Parse json tag
					if jsonTag, ok := structTags.Lookup("json"); ok {
						parts := strings.Split(jsonTag, ",")
						if len(parts) > 0 && parts[0] != "" {
							jsonName = parts[0]
						}
						tags["json"] = jsonTag
					}

					// Parse binding tag for required fields
					if bindingTag, ok := structTags.Lookup("binding"); ok {
						required = strings.Contains(bindingTag, "required")
						tags["binding"] = bindingTag
					}
				}

				// Extract field description from comments
				fieldDescription := ""
				if field.Doc != nil {
					for _, comment := range field.Doc.List {
						text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
						if fieldDescription != "" {
							fieldDescription += " "
						}
						fieldDescription += text
					}
				}

				// Create a new field
				requestField := RequestBodyField{
					Name:        fieldName,
					Type:        fieldType,
					JSONName:    jsonName,
					Required:    required,
					Description: fieldDescription,
					Tags:        tags,
				}

				fields = append(fields, requestField)
			}

			// Create the request body
			requestBody = &RequestBody{
				TypeName:    structName,
				Fields:      fields,
				Description: "",
			}

			return false // Stop inspecting once we've found our struct
		}
		return true
	})

	return requestBody, nil
}

// extractAnnotations extracts annotations from comments comments
func (p *Parser) extractAnnotations(comments *ast.CommentGroup) map[string]string {
	annotations := make(map[string]string)

	parsingDescription := false
	for _, comment := range comments.List {
		text := comment.Text

		// Extract @name
		if matches := namePattern.FindStringSubmatch(text); len(matches) > 1 {
			annotations["name"] = matches[1]
		}

		// Extract @route METHOD /path
		if matches := routePattern.FindStringSubmatch(text); len(matches) > 2 {
			annotations["route_method"] = matches[1]
			annotations["route_path"] = matches[2]
		}

		// Extract @body
		if matches := bodyPattern.FindStringSubmatch(text); len(matches) > 1 {
			annotations["body"] = matches[1]
		}

		// Extract @description
		descIndex := strings.Index(text, "@description")
		if descIndex != -1 || parsingDescription {
			parsingDescription = true

			var descText string
			// Get the text after @description
			if descIndex != -1 {
				descText = text[descIndex+len("@description"):]
			} else {
				descText = text
			}

			// Find the next annotation tag if there is one
			nextTagIndex := strings.IndexAny(descText, "@")

			if nextTagIndex != -1 {
				// Only take the text until the next tag
				parsingDescription = false
				descText = descText[:nextTagIndex]
			}

			descText = strings.TrimSpace(strings.ReplaceAll(descText, "/", ""))

			annotations["description"] += descText + "\n"
		}
	}

	return annotations
}
