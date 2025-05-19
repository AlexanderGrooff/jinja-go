#!/bin/bash
set -e

# Default branch to compare against is main/master
COMPARE_BRANCH=${1:-main}
CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)

# Create benchstat directory if it doesn't exist
mkdir -p benchstat

echo "Comparing benchmark performance: $CURRENT_BRANCH vs $COMPARE_BRANCH"

# Save current branch benchmarks
echo "Running benchmarks on current branch ($CURRENT_BRANCH)..."
go test ./pkg/ansiblejinja -bench=. -benchmem -count=5 > benchstat/current.txt

# Switch to compare branch and run benchmarks
echo "Switching to $COMPARE_BRANCH branch..."
git stash -q || true
git checkout $COMPARE_BRANCH
echo "Running benchmarks on $COMPARE_BRANCH branch..."
go test ./pkg/ansiblejinja -bench=. -benchmem -count=5 > benchstat/compare.txt

# Switch back to original branch
echo "Switching back to $CURRENT_BRANCH branch..."
git checkout $CURRENT_BRANCH
git stash pop -q 2>/dev/null || true

# Run benchstat comparison
echo "Benchmark comparison results:"
echo "============================"
benchstat benchstat/compare.txt benchstat/current.txt

# Cleanup
echo "Cleaning up temporary benchmark files..."
rm -f benchstat/current.txt benchstat/compare.txt

echo "Done!" 