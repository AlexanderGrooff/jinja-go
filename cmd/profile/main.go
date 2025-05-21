package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/AlexanderGrooff/jinja-go"
)

var (
	cpuprofile   = flag.String("cpuprofile", "", "write cpu profile to file")
	memprofile   = flag.String("memprofile", "", "write memory profile to file")
	blockprofile = flag.String("blockprofile", "", "write goroutine blocking profile to file")
	templateFile = flag.String("template", "", "template file to render")
	contextFile  = flag.String("context", "", "JSON file with context data")
	iterations   = flag.Int("iterations", 1000, "number of iterations to run")
	template     = flag.String("template-string", "", "template string to render (alternative to template file)")
	outputDir    = flag.String("output-dir", "profile", "directory to store profile output")
)

func main() {
	flag.Parse()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Get template
	var templateContent string
	if *templateFile != "" {
		content, err := ioutil.ReadFile(*templateFile)
		if err != nil {
			log.Fatalf("Failed to read template file: %v", err)
		}
		templateContent = string(content)
	} else if *template != "" {
		templateContent = *template
	} else {
		log.Fatal("Either --template or --template-string must be provided")
	}

	// Get context
	var context map[string]interface{}
	if *contextFile != "" {
		content, err := ioutil.ReadFile(*contextFile)
		if err != nil {
			log.Fatalf("Failed to read context file: %v", err)
		}
		if err := json.Unmarshal(content, &context); err != nil {
			log.Fatalf("Failed to parse context JSON: %v", err)
		}
	} else {
		context = make(map[string]interface{})
	}

	// Start CPU profiling if requested
	if *cpuprofile != "" {
		cpuFile := filepath.Join(*outputDir, *cpuprofile)
		f, err := os.Create(cpuFile)
		if err != nil {
			log.Fatalf("Failed to create CPU profile file: %v", err)
		}
		defer f.Close()

		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatalf("Failed to start CPU profile: %v", err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("CPU profiling enabled, writing to %s\n", cpuFile)
	}

	// Perform the template rendering
	fmt.Printf("Rendering template %d times\n", *iterations)
	start := time.Now()

	for i := 0; i < *iterations; i++ {
		result, err := jinja.TemplateString(templateContent, context)
		if err != nil {
			log.Fatalf("Failed to render template: %v", err)
		}
		// Use result to prevent compiler optimization
		if i == *iterations-1 {
			fmt.Printf("Result length: %d\n", len(result))
		}
	}

	duration := time.Since(start)
	fmt.Printf("Time taken: %v\n", duration)
	fmt.Printf("Average time per iteration: %v\n", duration/time.Duration(*iterations))

	// Memory profiling
	if *memprofile != "" {
		memFile := filepath.Join(*outputDir, *memprofile)
		f, err := os.Create(memFile)
		if err != nil {
			log.Fatalf("Failed to create memory profile file: %v", err)
		}
		defer f.Close()

		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatalf("Failed to write memory profile: %v", err)
		}
		fmt.Printf("Memory profile written to %s\n", memFile)
	}

	// Block profiling
	if *blockprofile != "" {
		blockFile := filepath.Join(*outputDir, *blockprofile)
		f, err := os.Create(blockFile)
		if err != nil {
			log.Fatalf("Failed to create block profile file: %v", err)
		}
		defer f.Close()

		if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
			log.Fatalf("Failed to write block profile: %v", err)
		}
		fmt.Printf("Block profile written to %s\n", blockFile)
	}
}
