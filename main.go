package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	// Parse command line arguments
	inputDir := flag.String("input", ".", "Directory containing Go handler code")
	outputDir := flag.String("output", "./bruno", "Directory for Bruno files")
	flag.Parse()

	// Create the parser that extracts annotated handlers
	parser := NewParser()

	// Parse the handler functions and struct definitions
	fmt.Printf("Scanning %s for annotated handlers...\n", *inputDir)
	routes, err := parser.ParseDirectory(*inputDir)
	if err != nil {
		fmt.Printf("Error parsing code: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d handlers with route annotations\n", len(routes))

	// Create the Bruno generator
	brunoGen := NewBrunoGenerator(*outputDir)

	// Generate Bruno files for each handler with route annotations
	for _, route := range routes {
		fmt.Printf("Processing handler: %s %s\n", route.Method, route.Path)

		// Generate Bruno .bru file
		if err := brunoGen.GenerateRequestFile(route); err != nil {
			fmt.Printf("Error generating Bruno file: %v\n", err)
			continue
		}
	}

	fmt.Printf("\nDone! Generated Bruno files in %s\n", *outputDir)
}
