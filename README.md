# Jinja Go

A Go library that mimics Jinja's templating behavior.

## Project Goals

This project aims to provide a reusable library with two main functions:

-   `TemplateString(template string, context map[string]interface{}) (string, error)`: Evaluates a Jinja-like template string. Variables in the format `{{ variable_name }}` are replaced with values from the context map.
-   `EvaluateExpression(expression string, context map[string]interface{}) (interface{}, error)`: Evaluates a Jinja-like expression string using the provided context. An error is returned if the expression cannot be evaluated.

Additionally, the library will support:

-   Built-in functions and filters comparable to those in Jinja (e.g., `lookup`, `urlencode`, `map`, `default`).
-   Basic flow control structures (e.g., `{% for item in items %}`, `{% if condition %}`).

## Usage

```go
package main

import (
	"fmt"

	"github.com/AlexanderGrooff/jinja-go"
)

func main() {
	context := map[string]interface{}{
		"name": "World",
		"isAdmin": true,
	}

	// TemplateString example
	templated, err := jinja.TemplateString("Hello {{ name }}!", context)
	if err != nil {
		fmt.Printf("TemplateString Error: %v\n", err)
		return
	}
	fmt.Println(templated) // Output: Hello World!

	// EvaluateExpression example
	isAdmin, err := jinja.EvaluateExpression("isAdmin", context)
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

## Profiling

In addition to benchmarking, the library includes profiling tools to identify performance bottlenecks and optimize critical sections of code.

### Quick Start

```bash
# Profile the complex_template with CPU, memory, and block profiling
make profile-complex

# Profile nested_loops template (one of the most performance-critical patterns)
make profile-nested-loops

# Profile all templates from the benchmark suite
make profile-all

# Run custom profiling
make profile ARGS="--template conditional --cpu --iterations 5000"
```

### Analyzing Profiles

After running a profile, analyze the results:

```bash
# Web-based visualization (most comprehensive)
go tool pprof -http=:8080 profile_results/complex_template/cpu.prof

# Text-based analysis
go tool pprof profile_results/template_name/cpu.prof
(pprof) top10                # Show top 10 functions by CPU usage
(pprof) list TemplateString  # Show time spent in function
```

For more detailed information on profiling and performance optimization guidelines, see [performance.md](performance.md).

### Cross-Language Benchmarks

You can directly compare this Go implementation against Python's Jinja2 and other Go-based Jinja-like libraries (such as Pongo2) using the cross-language benchmarking tools:

```bash
# Run with default settings
make cross-benchmark

# Run with custom iterations and output directory
./cmd/benchmark/run_benchmarks.sh --iterations 5000 --output-dir custom_benchmarks

# Run with custom template test cases
./cmd/benchmark/run_benchmarks.sh --templates path/to/custom_templates.json
```

The cross-benchmark tool:

1. Runs identical templates through both the Python and Go implementations (including other Go libraries like Pongo2)
2. Measures rendering time for each template
3. Calculates the speed difference between implementations
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

You can view the [latest comparison report](benchstat/cross/comparison_report.txt) to see the current performance differences between all implementations.

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
    - Dictionary methods like `.get()` (`dict.get('key', 'default')`)
    - String methods like `.format()` (`"Hello, {}!".format("world")`)
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
  - `lookup` filter with `file` and `env` sources

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
  - Additional common Jinja filters (`urlencode`, etc.)
  - Additional lookup plugin types for the `lookup` filter
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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

### Pre-commit Hooks

The pre-commit hooks will:
1. Run benchmarks before each commit
2. Compare with previous benchmark results 
3. Show performance changes

Install pre-commit hooks with:

```bash
pre-commit install
``` 