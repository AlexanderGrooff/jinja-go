# Performance Guidelines

Performance of this library is critical. Always ensure that the speed of the Jinja templating engine is as fast as possible.

## Benchmarking

Use the benchmark commands available in the Makefile to measure and compare performance:

```bash
# Run all benchmarks
make benchmark

# Run benchmarks and save results for comparison
make benchmark-save

# Compare with previous benchmark results
make benchmark-compare

# Generate a benchmark report
make benchmark-report

# Compare with Python's Jinja2 implementation
make cross-benchmark
```

## Profiling

The library includes a profiling tool to identify performance bottlenecks:

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

### Manual Profiling

You can run the profiler directly for more control:

```bash
# Build the profiler
go build -o profile_tool cmd/profile/main.go

# Run with custom options
./profile_tool --template-string="{{greeting}}, {{name}}!" \
  --context='{"greeting":"Hello","name":"World"}' \
  --iterations=10000 \
  --cpuprofile="cpu.prof" \
  --memprofile="mem.prof"
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

## Performance Optimization Patterns

When optimizing code, consider these patterns:

1. **Template Caching**: Templates should be parsed once and reused.
   - Already implemented in `TemplateString()` function.

2. **Reduce Allocations**:
   - Pre-allocate slices with reasonable capacity
   - Use object pools for frequently allocated objects
   - Avoid unnecessary string operations (prefer indexing over slicing)
   - Use `strings.Builder` with hint capacity

3. **Hot Paths to Optimize**:
   - `splitExpressionWithFilters`: High allocation, called for every expression
   - `parseControlTagDetail`: High overhead for control tags
   - `handleForStatement`: Significant overhead for nested loops
   - `Parser.ParseNext`: Core parsing function with recursive patterns

4. **Implementation Strategies**:
   - Consider adding a tokenization stage before parsing
   - Replace recursive descent with iterative approaches where possible
   - Cache expression evaluation results when appropriate
   - Consider compiling frequently used templates to Go functions

For more detailed recommendations, see `profile_results/performance_recommendations.md`. 