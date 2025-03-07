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

// Annotation patterns
var (
	routePattern       = regexp.MustCompile(`@route\s+([A-Z]+)\s+(.+)`)
	descriptionPattern = regexp.MustCompile(`@description\s+(.+)`)
	bodyPattern        = regexp.MustCompile(`@body\s+(\w+)`)
)

// Parser extracts information about API routes and structures
type Parser struct {
	structs map[string]*RequestBody
	routes  []Route
}

// NewParser creates a new Parser
func NewParser() *Parser {
	return &Parser{
		structs: make(map[string]*RequestBody),
		routes:  []Route{},
	}
}

// ParseDirectory parses all Go files in a directory
func (p *Parser) ParseDirectory(dirPath string) ([]Route, error) {
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			if err := p.ParseFile(path); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Associate request body structs with routes
	for i, route := range p.routes {
		if body, exists := p.structs[route.BodyType]; exists && body != nil {
			p.routes[i].RequestBody = body
		}
	}

	return p.routes, nil
}

// ParseFile parses a single Go file for annotations and struct definitions
func (p *Parser) ParseFile(filePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// Extract struct definitions and handler annotations
	ast.Inspect(node, func(n ast.Node) bool {
		// Look for struct definitions
		if typeSpec, ok := n.(*ast.TypeSpec); ok {
			p.processTypeSpec(typeSpec, fset)
			return true
		}

		// Look for function declarations (handlers)
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			p.processFuncDecl(funcDecl, fset)
			return true
		}

		return true
	})

	return nil
}

// processTypeSpec processes a type declaration to extract struct information
func (p *Parser) processTypeSpec(typeSpec *ast.TypeSpec, fset *token.FileSet) {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return // Not a struct
	}

	structName := typeSpec.Name.Name

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

	// Create the request body and store in map
	requestBody := &RequestBody{
		TypeName:    structName,
		Fields:      fields,
		Description: "", // We don't need struct descriptions as we focus on handlers
	}

	p.structs[structName] = requestBody
}

// processFuncDecl processes a function declaration to extract handler annotations
func (p *Parser) processFuncDecl(funcDecl *ast.FuncDecl, fset *token.FileSet) {
	// Skip if no comments
	if funcDecl.Doc == nil {
		return
	}

	handlerName := funcDecl.Name.Name

	// Extract annotations from comments
	annotations := p.extractAnnotations(funcDecl.Doc)

	// Check if we have route information
	method, hasMethod := annotations["route_method"]
	path, hasPath := annotations["route_path"]

	// Only process functions with a @route annotation
	if hasMethod && hasPath {
		// Create a new route from handler annotations
		route := Route{
			Method:      method,
			Path:        path,
			Handler:     handlerName,
			Description: annotations["description"],
			BodyType:    annotations["body"],
			Tags:        make(map[string]string),
		}

		p.routes = append(p.routes, route)
		fmt.Printf("Found route: %s %s in handler %s\n", method, path, handlerName)
	}
}

// extractAnnotations extracts annotations from comment groups
func (p *Parser) extractAnnotations(comments *ast.CommentGroup) map[string]string {
	result := make(map[string]string)

	if comments == nil {
		return result
	}

	// Convert the comment group to a single string
	var fullComment strings.Builder
	for _, comment := range comments.List {
		// Clean the comment by removing the comment markers
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		fullComment.WriteString(text + "\n")
	}

	commentText := fullComment.String()

	// Extract @route (should be a single line)
	if matches := routePattern.FindStringSubmatch(commentText); len(matches) > 2 {
		result["route_method"] = matches[1]
		result["route_path"] = matches[2]
	}

	// Extract @body (should be a single line)
	if matches := bodyPattern.FindStringSubmatch(commentText); len(matches) > 1 {
		result["body"] = matches[1]
	}

	// Extract @description (can be multi-line)
	descIndex := strings.Index(commentText, "@description")
	if descIndex != -1 {
		// Get the text after @description
		descText := commentText[descIndex+len("@description"):]

		// Find the next annotation tag if there is one
		nextTagIndex := strings.IndexAny(descText, "@")

		if nextTagIndex != -1 {
			// Only take the text until the next tag
			descText = descText[:nextTagIndex]
		}

		// Clean up the description text
		description := strings.TrimSpace(descText)
		// Replace newlines and excessive spaces with a single space
		description = regexp.MustCompile(`\s+`).ReplaceAllString(description, " ")

		result["description"] = description
	}

	return result
}
