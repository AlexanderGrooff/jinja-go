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

	"github.com/AlexanderGrooff/ansible-jinja-go"
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

# Run cross-language benchmarks against Python's Jinja2
make cross-benchmark
```

The repository uses [benchstat](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat) to compare benchmark results, and pre-commit hooks automatically run benchmarks and compare with previous results.

### Cross-Language Benchmarks

You can directly compare this Go implementation against Python's Jinja2 using the cross-language benchmarking tools:

```bash
# Run with default settings
make cross-benchmark

# Run with custom iterations and output directory
./cmd/benchmark/run_benchmarks.sh --iterations 5000 --output-dir custom_benchmarks

# Run with custom template test cases
./cmd/benchmark/run_benchmarks.sh --templates path/to/custom_templates.json
```

The cross-benchmark tool:

1. Runs identical templates through both the Go and Python implementations
2. Measures rendering time for each template
3. Calculates the speed difference (how many times faster Go is)
4. Generates a detailed comparison report

Custom template test cases can be defined in a JSON file following this format:

```json
[
  {
    "name": "template_name",
    "template": "Hello, {{ name }}!",
    "context": {"name": "World"}
  },
  // More test cases...
]
```

The comparison provides insight into performance characteristics of both implementations, which is useful for:
- Identifying areas where the Go implementation can be optimized
- Quantifying performance gains for various template features
- Tracking performance improvements over time

You can view the [latest comparison report](benchstat/cross/comparison_report.txt) to see the current performance difference between the Go and Python implementations.

## Implementation Status

### Already Implemented Features

- **Template Syntax**
  - Basic variable substitution (`{{ variable }}`)
  - Comments (`{# comment #}`)
  - Conditional statements (`{% if %}`, `{% elif %}`, `{% else %}`, `{% endif %}`)
  - Loop structures (`{% for item in items %}`, `{% endfor %}`) with loop variable support

- **Expression Evaluation**
  - Basic literals (integers, floats, strings, booleans, null/None)
  - Variable access and context lookup
  - Pythonic data types:
    - Lists (`[1, 2, 3]`)
    - Dictionaries (`{'key': 'value'}`)
  - Object/attribute access (`object.attribute`)
  - Subscript access (`array[index]`, `dict['key']`, negative indices)
  - LALR (Look-Ahead LR) parser for robust expression evaluation
    - Improved parsing performance and reliability
    - Proper operator precedence handling
    - Support for complex expressions such as `a * (b + c) / d`
  - Complex nested expression handling with multiple subscript operations
  - Basic filters (e.g., `{{ var | default('fallback') }}`)

- **Operators**
  - Arithmetic operators (`+`, `-`, `*`, `/`, `//` (floor division), `%` (modulo), `**` (power))
  - Unary operators (`not`, `-`, `+`)
  - Comparison operators (`==`, `!=`, `>`, `<`, `>=`, `<=`)
  - Logical operators (`and`, `or`) with short-circuit evaluation
  - Identity operators (`is`, `is not`)
  - Membership operators (`in`)
  - String operations (concatenation, repetition)

- **Filters**
  - `default` filter
  - `join` filter
  - `upper` filter
  - `lower` filter
  - `capitalize` filter
  - `replace` filter
  - `trim` filter
  - `list` filter
  - `escape` filter
  - `map` filter
  - `items` filter

### Planned Features

- **Template Syntax**
  - Include support (`{% include 'page.html' %}`)
  - Macro definitions (`{% macro %}`/`{% endmacro %}`)
  - Block and extends for template inheritance (`{% block %}`, `{% extends %}`)
  - Set statements (`{% set %}`)
  - With blocks (`{% with %}`)
  - Loop controls (`{% break %}`, `{% continue %}`)
  - Whitespace control (using `-` in tags like `{%-` and `-%}`)
  - Expression statements (`{% do expression %}`)
  - Debug statements (`{% debug %}`)

- **Expression Evaluation**
  - More filters (e.g., `{{ url | urlencode }}`)
  - Tests (`{{ user is defined }}`, `{{ user is not none }}`)
  - String formatting and f-strings
  - List comprehensions
  - Generator expressions (iterables)
  - Auto-escaping support

- **Control Structures**
  - More complex control structures

- **Filters and Functions**
  - Additional common Ansible Jinja filters (`urlencode`, etc.)
  - Lookup plugin support
  - More built-in functions
  - Complete set of built-in tests (`defined`, `none`, `iterable`, etc.)
  - Translation/internationalization support (gettext)

- **Advanced Features**
  - Macro definitions
  - Include/import functionality
  - Block/extends for template inheritance
  - Context scoping and namespaces
  - Custom tests and filters
  - Auto-escaping configuration

### Broader improvements to be made

1. Error handling improvements - standardize error reporting across modules
1. Handling of edge cases in string literals and escaping
1. Better documentation of supported features

### Pre-commit Hooks

The pre-commit hooks will:
1. Run benchmarks before each commit
2. Compare with previous benchmark results 
3. Show performance changes

Install pre-commit hooks with:

```bash
pre-commit install
``` 