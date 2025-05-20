#!/usr/bin/env python3

import json
import argparse
import sys
from tabulate import tabulate

def main():
    parser = argparse.ArgumentParser(description="Compare Go and Python Jinja benchmark results")
    parser.add_argument(
        "--go-results", type=str, default="go_results.json",
        help="Path to Go benchmark results JSON file"
    )
    parser.add_argument(
        "--python-results", type=str, default="python_results.json",
        help="Path to Python benchmark results JSON file"
    )
    parser.add_argument(
        "--output", type=str, default=None,
        help="Optional output file for the comparison (defaults to stdout)"
    )
    args = parser.parse_args()

    # Load results
    try:
        with open(args.go_results, 'r') as f:
            go_results = json.load(f)
    except Exception as e:
        print(f"Error loading Go results: {e}", file=sys.stderr)
        return 1

    try:
        with open(args.python_results, 'r') as f:
            python_results = json.load(f)
    except Exception as e:
        print(f"Error loading Python results: {e}", file=sys.stderr)
        return 1

    # Convert to dictionaries for easier lookup
    go_dict = {item["name"]: item["execution_time_ms"] for item in go_results}
    python_dict = {item["name"]: item["execution_time_ms"] for item in python_results}

    # Get all unique benchmark names
    all_names = sorted(set(list(go_dict.keys()) + list(python_dict.keys())))

    # Prepare comparison table
    comparison = []
    for name in all_names:
        go_time = go_dict.get(name, "N/A")
        python_time = python_dict.get(name, "N/A")
        
        # Calculate speedup ratio when we have both measurements
        if go_time != "N/A" and python_time != "N/A":
            speedup = python_time / go_time
            speedup_str = f"{speedup:.2f}x"
        else:
            speedup_str = "N/A"
        
        comparison.append([
            name,
            f"{go_time:.6f}" if go_time != "N/A" else "N/A",
            f"{python_time:.6f}" if python_time != "N/A" else "N/A",
            speedup_str
        ])

    # Create the table
    headers = ["Benchmark", "Go (ms)", "Python (ms)", "Speedup (Go vs Python)"]
    table = tabulate(comparison, headers=headers, tablefmt="grid")
    
    # Add summary
    valid_comparisons = [row for row in comparison if row[3] != "N/A"]
    if valid_comparisons:
        speedups = [float(row[3].replace('x', '')) for row in valid_comparisons]
        avg_speedup = sum(speedups) / len(speedups)
        summary = f"\nSummary:\nAverage speedup: {avg_speedup:.2f}x (higher means Go implementation is faster)"
    else:
        summary = "\nNo valid comparisons found"
    
    result = table + summary

    # Output results
    if args.output:
        with open(args.output, 'w') as f:
            f.write(result)
        print(f"Comparison written to {args.output}")
    else:
        print(result)
    
    return 0

if __name__ == "__main__":
    sys.exit(main()) 