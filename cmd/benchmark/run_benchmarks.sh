#!/usr/bin/env bash

set -e

# Define default parameters
ITERATIONS=100000
OUTPUT_DIR="benchmark_results"
GO_RESULTS="$OUTPUT_DIR/go_results.json"
PYTHON_RESULTS="$OUTPUT_DIR/python_results.json"
PONGO2_RESULTS="$OUTPUT_DIR/pongo2_results.json"
COMPARISON_REPORT="$OUTPUT_DIR/comparison_report.txt"
TEMPLATES_FILE="cmd/benchmark/templates.json"
VERBOSE="false"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --iterations)
      ITERATIONS="$2"
      shift 2
      ;;
    --output-dir)
      OUTPUT_DIR="$2"
      GO_RESULTS="$OUTPUT_DIR/go_results.json"
      PYTHON_RESULTS="$OUTPUT_DIR/python_results.json"
      PONGO2_RESULTS="$OUTPUT_DIR/pongo2_results.json"
      COMPARISON_REPORT="$OUTPUT_DIR/comparison_report.txt"
      shift 2
      ;;
    --templates)
      TEMPLATES_FILE="$2"
      shift 2
      ;;
    --verbose)
      VERBOSE="true"
      shift
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Create output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

echo "Running benchmarks with $ITERATIONS iterations..."
echo "Using templates from: $TEMPLATES_FILE"

# Check if Go is installed
if ! command -v go &> /dev/null; then
  echo "Error: Go is not installed. Please install Go to run the Go benchmarks."
  exit 1
fi

# Make sure pongo2 benchmark module has its dependencies
echo "Setting up benchmark dependencies..."
(cd cmd/benchmark/pongo2_benchmark && go mod tidy)

# Create directory for pongo2 benchmark if it doesn't exist
mkdir -p cmd/benchmark/pongo2_benchmark

# Build and run Go benchmarks
echo "Building Go benchmarking tool..."
go build -o "$OUTPUT_DIR/go_benchmark" cmd/benchmark/main.go

echo "Running Go benchmarks..."
"$OUTPUT_DIR/go_benchmark" --iterations "$ITERATIONS" --output "$GO_RESULTS" --templates "$TEMPLATES_FILE"

# Build and run Pongo2 benchmarks
echo "Building Pongo2 benchmarking tool..."
(cd cmd/benchmark/pongo2_benchmark && go build -o "../../../$OUTPUT_DIR/pongo2_benchmark" .)

echo "Running Pongo2 benchmarks..."
"$OUTPUT_DIR/pongo2_benchmark" --iterations "$ITERATIONS" --output "$PONGO2_RESULTS" --templates "$TEMPLATES_FILE"

# Check if Python and required packages are installed
if ! command -v python3 &> /dev/null; then
  echo "Error: Python 3 is not installed. Please install Python 3 to run the Python benchmarks."
  exit 1
fi

# Function to install Python packages if needed
install_python_package() {
  local package=$1
  if ! python3 -c "import $package" &> /dev/null; then
    echo "Installing Python $package package..."
    pip3 install $package
  fi
}

# Install required Python packages
install_python_package "jinja2"
install_python_package "tabulate"

# Run Python benchmarks
echo "Running Python Jinja2 benchmarks..."
chmod +x cmd/benchmark/python_benchmark.py
cmd/benchmark/python_benchmark.py --iterations "$ITERATIONS" --output "$PYTHON_RESULTS" --templates "$TEMPLATES_FILE"

# Generate comparison report
echo "Generating comparison report..."
chmod +x cmd/benchmark/compare_results.py
cmd/benchmark/compare_results.py --go-results "$GO_RESULTS" --python-results "$PYTHON_RESULTS" --pongo2-results "$PONGO2_RESULTS" --output "$COMPARISON_REPORT"

echo "Done! Results saved to:"
echo "  Go results: $GO_RESULTS"
echo "  Python results: $PYTHON_RESULTS"
echo "  Pongo2 results: $PONGO2_RESULTS"
echo "  Comparison report: $COMPARISON_REPORT"

# Display the report
echo ""
echo "Benchmark Results Comparison:"
echo "============================"
cat "$COMPARISON_REPORT"

# Additional benchmark suggestions
echo ""
echo "For additional benchmarking options:"
echo "1. Run with more iterations for more stable results: --iterations 10000"
echo "2. Create custom template files for specific use cases"
echo "3. Try with different template engines (e.g., Jinja2, Pongo2, fasttemplate)"
echo "4. Compare with different versions of Go and Python" 