package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/flosch/pongo2/v6"
)

type BenchmarkCase struct {
	Name     string                 `json:"name"`
	Template string                 `json:"template"`
	Context  map[string]interface{} `json:"context"`
}

type BenchmarkResult struct {
	Name            string  `json:"name"`
	ExecutionTimeMs float64 `json:"execution_time_ms"`
}

func main() {
	iterations := flag.Int("iterations", 1000, "Number of iterations for each benchmark")
	outputFile := flag.String("output", "pongo2_results.json", "Output file for benchmark results")
	templatesFile := flag.String("templates", "cmd/benchmark/templates.json", "JSON file containing template test cases")
	flag.Parse()

	// Load benchmark cases from JSON file
	benchmarks, err := loadBenchmarkCases(*templatesFile)
	if err != nil {
		fmt.Printf("Error loading template cases: %v\n", err)
		os.Exit(1)
	}

	results := make([]BenchmarkResult, 0, len(benchmarks))

	// Run benchmarks
	for _, bm := range benchmarks {
		fmt.Printf("Running benchmark: %s\n", bm.Name)
		startTime := time.Now()

		// Compile template once
		tpl, err := pongo2.FromString(bm.Template)
		if err != nil {
			fmt.Printf("Error compiling template for benchmark %s: %v\n", bm.Name, err)
			continue
		}

		for i := 0; i < *iterations; i++ {
			_, err := tpl.Execute(pongo2.Context(bm.Context))
			if err != nil {
				fmt.Printf("Error in benchmark %s: %v\n", bm.Name, err)
				break
			}
		}

		elapsed := time.Since(startTime)
		avgTimeMs := float64(elapsed.Microseconds()) / float64(*iterations) / 1000.0

		results = append(results, BenchmarkResult{
			Name:            bm.Name,
			ExecutionTimeMs: avgTimeMs,
		})

		fmt.Printf("  Average time: %.6f ms\n", avgTimeMs)
	}

	// Write results to file
	jsonData, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling results: %v\n", err)
		os.Exit(1)
	}

	err = os.WriteFile(*outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing results to file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Benchmark results written to %s\n", *outputFile)
}

// loadBenchmarkCases loads benchmark test cases from a JSON file
func loadBenchmarkCases(filename string) ([]BenchmarkCase, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read templates file: %w", err)
	}

	var benchmarks []BenchmarkCase
	err = json.Unmarshal(data, &benchmarks)
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates JSON: %w", err)
	}

	return benchmarks, nil
}
