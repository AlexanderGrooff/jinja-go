# Benchmark Statistics

This directory contains benchmark results for the Jinja Go implementation.

The files in this directory:
- `previous.txt`: Previous benchmark results stored for comparison
- `latest.txt`: Current benchmark results (not committed to git)

These files are used by the pre-commit hooks to:
1. Run benchmarks and store results
2. Compare results with previous runs
3. Track performance changes between commits

This helps maintain performance standards across changes to the codebase. 