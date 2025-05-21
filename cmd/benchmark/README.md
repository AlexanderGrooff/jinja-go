# Jinja Templating Benchmarks

This directory contains benchmarking tools to compare the performance of different Jinja templating implementations:

1. Our Go implementation (jinja-go)
2. Python's Jinja2 implementation
3. Pongo2 (Go implementation)

## Module Structure

This benchmark setup uses separate Go modules to isolate dependencies:

- The main jinja-go library has its own go.mod in the project root
- The Pongo2 benchmark has its own go.mod in the `pongo2_benchmark/` directory

This separation prevents benchmark dependencies from polluting the main project's dependency tree.

## Running Benchmarks

You can run the benchmarks using the Makefile commands:

```bash
# Compare with Python's Jinja2
make cross-benchmark

# Compare with other Go template libraries
make golang-jinja-compare
```

Or directly using the benchmark script:

```bash
./cmd/benchmark/run_benchmarks.sh --iterations 100000 --output-dir benchmark_results
```

## Command Line Options

The benchmark script accepts these options:

- `--iterations N`: Number of iterations for each benchmark (default: 100000)
- `--output-dir DIR`: Directory to store results (default: benchmark_results)
- `--templates FILE`: JSON file with template test cases (default: cmd/benchmark/templates.json)
- `--verbose`: Enable verbose output

## Adding Custom Templates

To add custom templates for benchmarking, edit the `templates.json` file. Each benchmark case requires:

```json
{
  "name": "benchmark_name",
  "template": "Template with {{ variables }} and {% logic %}",
  "context": {"variables": "to", "use": "in template"}
}
```

## Implementation Differences

Note that different templating engines have varying levels of support for Jinja syntax features:

- **jinja-go**: Full compatibility with Ansible's Jinja2 syntax (goal of this project)
- **Python Jinja2**: The reference implementation
- **Pongo2**: A Go implementation of the Django template engine (similar to Jinja2 but with some differences):
  - Different filter syntax: `{{ variable|filter:"arg" }}` vs Jinja's `{{ variable|filter("arg") }}`
  - Some filters have different names or don't exist
  - Slightly different loop handling

When benchmarking, some templates may fail with specific engines due to these differences.

## Adding More Template Engines

To add support for benchmarking with additional template engines:

1. Create a new directory under `cmd/benchmark/` (e.g., `fasttemplate_benchmark/`)
2. Initialize a separate Go module in that directory (`go mod init ...`)
3. Implement a benchmark program similar to the existing ones
4. Update `run_benchmarks.sh` to include the new template engine
5. Update `compare_results.py` to include the new results in the comparison

By using separate modules for each templating engine, you keep their dependencies isolated from the main project.

## Benchmark Results

The benchmark results are stored in:
- `benchmark_results/go_results.json` - Our library
- `benchmark_results/python_results.json` - Python Jinja2
- `benchmark_results/pongo2_results.json` - Pongo2
- `benchmark_results/comparison_report.txt` - Detailed comparison

## Interpreting Results

The comparison report shows:
- Execution time in milliseconds for each template engine
- Ratios between engines (higher means slower)
- Average performance ratios across all templates 