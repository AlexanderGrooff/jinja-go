# Ansible Jinja Go

A Go library that mimics Ansible's Jinja templating behavior.

## Project Goals

This project aims to provide a reusable library with two main functions:

-   `TemplateString(template string, context map[string]interface{}) (string, error)`: Evaluates a Jinja-like template string. Variables in the format `{{ variable_name }}` are replaced with values from the context map.
-   `EvaluateExpression(expression string, context map[string]interface{}) (interface{}, error)`: Evaluates a Jinja-like expression string using the provided context. An error is returned if the expression cannot be evaluated.

Additionally, the library will support:

-   Built-in functions and filters comparable to those in Ansible's Jinja (e.g., `lookup`, `urlencode`, `map`, `default`).
-   Basic flow control structures (e.g., `{% for item in items %}`, `{% if condition %}`).

## Usage

```go
package main

import (
	"fmt"

	"github.com/AlexanderGrooff/ansible-jinja-go/pkg/ansiblejinja"
)

func main() {
	context := map[string]interface{}{
		"name": "World",
		"isAdmin": true,
	}

	// TemplateString example
	templated, err := ansiblejinja.TemplateString("Hello {{ name }}!", context)
	if err != nil {
		fmt.Printf("TemplateString Error: %v\n", err)
		return
	}
	fmt.Println(templated) // Output: Hello World!

	// EvaluateExpression example
	isAdmin, err := ansiblejinja.EvaluateExpression("isAdmin", context)
	if err != nil {
		fmt.Printf("EvaluateExpression Error: %v\n", err)
		return
	}
	fmt.Printf("Is Admin: %v\n", isAdmin) // Output: Is Admin: true
} 
```

## Benchmarking

Performance is critical for this library. We use benchmarking to ensure that changes don't negatively impact performance.

### Running Benchmarks

```bash
# Run benchmarks without saving results
make benchmark

# Run benchmarks and save as latest
make benchmark-save

# Compare latest benchmarks with previous
make benchmark-compare

# Save latest as the new previous (baseline)
make benchmark-save-as-previous

# Compare with another branch
make benchmark-branch branch=main

# Generate and save a benchmark report
make benchmark-report
```

The repository uses [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) to compare benchmark results, and pre-commit hooks automatically run benchmarks and compare with previous results.

### Pre-commit Hooks

The pre-commit hooks will:
1. Run benchmarks before each commit
2. Compare with previous benchmark results 
3. Show performance changes

Install pre-commit hooks with:

```bash
pre-commit install
``` 