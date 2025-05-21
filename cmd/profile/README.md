# Jinja Template Engine Profiler

This tool helps identify performance bottlenecks in the Jinja template engine implementation.

## Quick Start

Use the Makefile targets to run common profiling scenarios:

```bash
# Profile the complex_template with all profiling types
make profile-complex

# Profile the nested_loops template with all profiling types
make profile-nested-loops 

# Profile all templates from the benchmark suite
make profile-all

# Run custom profiling
make profile ARGS="--template conditional --cpu --iterations 5000"
```

## Manual Usage

You can also run the profile script directly:

```bash
./cmd/profile/profile.sh --template template_name [options]
```

### Options

- `--template NAME`: Profile a specific template from templates.json
- `--iterations N`: Number of iterations to run (default: 10000)
- `--cpu`: Enable CPU profiling
- `--mem`: Enable memory profiling
- `--block`: Enable block profiling
- `--all`: Enable all profiling types
- `--all-templates`: Run profiling for all templates
- `--help`: Show help message

## Analyzing Results

Profiling results are saved to the `profile_results/` directory. For each template, you'll find:

- `template.txt`: The template that was profiled
- `context.json`: The context data used
- `profile_output.txt`: Text output from the profiling run
- `cpu.prof`, `mem.prof`, `block.prof`: Profile data files (if enabled)

### Visualizing with pprof

To visualize the results in a web browser:

```bash
go tool pprof -http=:8080 profile_results/template_name/cpu.prof
```

This opens a web UI at http://localhost:8080 where you can explore:

- Top functions (CPU time, memory allocations)
- Flame graphs
- Call graphs
- Source code view with hotspots

### Text-based Analysis

For text-based analysis:

```bash
go tool pprof profile_results/template_name/cpu.prof

# At the pprof prompt, try these commands:
(pprof) top10                # Show top 10 CPU-consuming functions
(pprof) list TemplateString  # Show source with time spent in the TemplateString function
(pprof) list Parser.ParseNext # Show source with time spent in the ParseNext function
```

## Common Performance Issues

When analyzing profiles, look for:

1. **Excessive Allocations**: Check memory profiles for functions making many allocations.
2. **String Concatenation**: Look for time spent in string handling operations.
3. **Recursive Parser Calls**: Check for deep call stacks in parsing functions.
4. **Repeated Template Parsing**: Templates should ideally be parsed once and cached.
5. **Expression Evaluation**: Check for inefficient expression evaluation.

## Optimization Opportunities

Based on profiling results, consider these optimization strategies:

1. **Cached templates**: Implement template caching to avoid repeated parsing.
2. **Optimized parsing**: Identify and optimize hotspots in the parser.
3. **Reduced allocations**: Reuse buffers and preallocate when possible.
4. **Switch to token-based parsing**: For complex templates, a tokenization step before parsing may be more efficient.
5. **Use sync.Pool**: For frequently allocated objects (nodes, parsers, etc.).
6. **Context optimizations**: Optimize variable lookups in context maps. 