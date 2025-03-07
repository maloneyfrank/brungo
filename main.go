package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	initializeLogging()
	logger := getLogger()
	inputDir := flag.String("input", ".", "Directory containing Go handler code")
	outputDir := flag.String("output", "./bruno", "Directory for Bruno files")
	flag.Parse()

	// Create the parser that extracts annotated handlers
	parser := NewParser()

	// Parse the handler functions and struct definitions
	logger.Info(fmt.Sprintf("Scanning %s for annotated handlers...", *inputDir))
	routes, err := parser.ParseDirectory(*inputDir)
	if err != nil {
		logger.Error(fmt.Sprintf("Error parsing code: %v", err))
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("Found %d handlers with route annotations", len(routes)))

	// Create the Bruno generator
	brunoGen := NewBrunoGenerator(*outputDir, "api.example.com")

	// Generate Bruno files for each handler with route annotations
	for _, route := range routes {
		logger.Info(fmt.Sprintf("Processing handler: %s %s", route.Method, route.Path))
		// Generate Bruno .bru file
		if err := brunoGen.GenerateRequestFile(route); err != nil {
			logger.Error(fmt.Sprintf("Error generating Bruno file: %v", err))
			continue
		}
	}
	logger.Info(fmt.Sprintf("\nDone! Generated Bruno files in %s", *outputDir))
}
