#!/usr/bin/env python3

import argparse
import json
import time
import jinja2

def main():
    parser = argparse.ArgumentParser(description="Benchmark Jinja2 templating")
    parser.add_argument(
        "--iterations", type=int, default=1000,
        help="Number of iterations for each benchmark"
    )
    parser.add_argument(
        "--output", type=str, default="python_results.json",
        help="Output file for benchmark results"
    )
    parser.add_argument(
        "--templates", type=str, default="cmd/benchmark/templates.json",
        help="JSON file containing template test cases"
    )
    args = parser.parse_args()
    
    # Load benchmark cases from JSON file
    try:
        with open(args.templates, 'r') as f:
            benchmarks = json.load(f)
    except Exception as e:
        print(f"Error loading template cases: {e}")
        return 1
    
    results = []
    
    # Set up Jinja2 environment
    env = jinja2.Environment(
        loader=jinja2.BaseLoader(),
        autoescape=jinja2.select_autoescape()
    )
    
    # Run benchmarks
    for bm in benchmarks:
        print(f"Running benchmark: {bm['name']}")
        
        # Compile the template first (this is typically cached in real usage)
        template = env.from_string(bm["template"])
        
        # Measure rendering time
        start_time = time.time()
        for i in range(args.iterations):
            result = template.render(**bm["context"])
        end_time = time.time()
        
        elapsed_ms = (end_time - start_time) * 1000 / args.iterations
        
        results.append({
            "name": bm["name"],
            "execution_time_ms": elapsed_ms
        })
        
        print(f"  Average time: {elapsed_ms:.6f} ms")
    
    # Write results to file
    with open(args.output, 'w') as f:
        json.dump(results, f, indent=2)
    
    print(f"Benchmark results written to {args.output}")
    return 0

if __name__ == "__main__":
    main() 