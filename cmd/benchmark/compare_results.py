#!/usr/bin/env python3

import json
import argparse
import sys
from tabulate import tabulate

def main():
    parser = argparse.ArgumentParser(description="Compare Go, Python Jinja, and Pongo2 benchmark results")
    parser.add_argument(
        "--go-results", type=str, default="go_results.json",
        help="Path to Go benchmark results JSON file"
    )
    parser.add_argument(
        "--python-results", type=str, default="python_results.json",
        help="Path to Python benchmark results JSON file"
    )
    parser.add_argument(
        "--pongo2-results", type=str, default="pongo2_results.json",
        help="Path to Pongo2 benchmark results JSON file"
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
        
    try:
        with open(args.pongo2_results, 'r') as f:
            pongo2_results = json.load(f)
    except Exception as e:
        print(f"Error loading Pongo2 results: {e}", file=sys.stderr)
        return 1

    # Convert to dictionaries for easier lookup
    go_dict = {item["name"]: item["execution_time_ms"] for item in go_results}
    python_dict = {item["name"]: item["execution_time_ms"] for item in python_results}
    pongo2_dict = {item["name"]: item["execution_time_ms"] for item in pongo2_results}

    # Get all unique benchmark names
    all_names = sorted(set(list(go_dict.keys()) + list(python_dict.keys()) + list(pongo2_dict.keys())))

    # Prepare comparison table
    comparison = []
    for name in all_names:
        go_time = go_dict.get(name, "N/A")
        python_time = python_dict.get(name, "N/A")
        pongo2_time = pongo2_dict.get(name, "N/A")
        
        # Calculate speedup ratios when we have both measurements
        if go_time != "N/A" and python_time != "N/A":
            py_speedup = python_time / go_time
            py_speedup_str = f"{py_speedup:.2f}x"
        else:
            py_speedup_str = "N/A"
            
        if go_time != "N/A" and pongo2_time != "N/A":
            pongo2_vs_go = pongo2_time / go_time
            pongo2_vs_go_str = f"{pongo2_vs_go:.2f}x"
        else:
            pongo2_vs_go_str = "N/A"
        
        comparison.append([
            name,
            f"{go_time:.6f}" if go_time != "N/A" else "N/A",
            f"{python_time:.6f}" if python_time != "N/A" else "N/A",
            f"{pongo2_time:.6f}" if pongo2_time != "N/A" else "N/A",
            py_speedup_str,
            pongo2_vs_go_str
        ])

    # Create the table
    headers = ["Benchmark", "Go (ms)", "Python (ms)", "Pongo2 (ms)", "Python/Go", "Pongo2/Go"]
    table = tabulate(comparison, headers=headers, tablefmt="grid")
    
    # Add summary
    python_comparisons = [float(row[4].replace('x', '')) for row in comparison if row[4] != "N/A"]
    pongo2_comparisons = [float(row[5].replace('x', '')) for row in comparison if row[5] != "N/A"]
    
    summary = "\nSummary:"
    if python_comparisons:
        avg_python_speedup = sum(python_comparisons) / len(python_comparisons)
        summary += f"\nAverage Python/Go ratio: {avg_python_speedup:.2f}x (higher means Python is slower)"
    else:
        summary += "\nNo valid Python/Go comparisons found"
        
    if pongo2_comparisons:
        avg_pongo2_speedup = sum(pongo2_comparisons) / len(pongo2_comparisons)
        summary += f"\nAverage Pongo2/Go ratio: {avg_pongo2_speedup:.2f}x (higher means Pongo2 is slower)"
    else:
        summary += "\nNo valid Pongo2/Go comparisons found"
    
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