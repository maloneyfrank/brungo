package main

import (
	"flag"
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
	logger.Error("Scanning %s for annotated handlers...\n", *inputDir)
	routes, err := parser.ParseDirectory(*inputDir)
	if err != nil {
		logger.Error("Error parsing code: %v\n", err)
		os.Exit(1)
	}

	logger.Info("Found %d handlers with route annotations\n", len(routes))

	// Create the Bruno generator
	brunoGen := NewBrunoGenerator(*outputDir)

	// Generate Bruno files for each handler with route annotations
	for _, route := range routes {
		logger.Error("Processing handler: %s %s\n", route.Method, route.Path)

		// Generate Bruno .bru file
		if err := brunoGen.GenerateRequestFile(route); err != nil {
			logger.Error("Error generating Bruno file: %v\n", err)
			continue
		}
	}

	logger.Info("\nDone! Generated Bruno files in %s\n", *outputDir)
}
