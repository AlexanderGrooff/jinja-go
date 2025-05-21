.PHONY: test benchmark benchmark-save benchmark-compare benchmark-report cross-benchmark golang-jinja-compare

test:
	go test ./

benchmark:
	go test ./ -bench=. -benchmem

# Run benchmarks and save as latest
benchmark-save:
	mkdir -p benchstat
	go test ./ -bench=. -benchmem -count=1 | tee benchstat/latest.txt
	@echo "Benchmark results saved to benchstat/latest.txt"

# Compare with previous benchmark
benchmark-compare:
	@if [ ! -f benchstat/previous.txt ]; then \
		echo "No previous benchmark found. Run 'make benchmark-save-as-previous' first."; \
		exit 1; \
	fi
	@if [ ! -f benchstat/latest.txt ]; then \
		echo "No latest benchmark found. Run 'make benchmark-save' first."; \
		exit 1; \
	fi
	benchstat benchstat/previous.txt benchstat/latest.txt

# Save latest as previous for future comparisons
benchmark-save-as-previous:
	@if [ ! -f benchstat/latest.txt ]; then \
		echo "No latest benchmark found. Run 'make benchmark-save' first."; \
		exit 1; \
	fi
	cp benchstat/latest.txt benchstat/previous.txt
	@echo "Latest benchmark saved as previous for future comparisons"

# Compare with another branch
benchmark-branch:
	@if [ -z "$(branch)" ]; then \
		echo "Usage: make benchmark-branch branch=<branch_name>"; \
		exit 1; \
	fi
	./scripts/compare_benchmarks.sh $(branch)

# Generate a benchmark report
benchmark-report:
	@if [ ! -f benchstat/previous.txt ]; then \
		echo "No previous benchmark file found"; \
		exit 1; \
	fi
	@mkdir -p benchstat/reports
	@REPORT_FILE="benchstat/reports/report-$$(date +%Y%m%d-%H%M%S).md"; \
	echo "# Benchmark Report - $$(date '+%Y-%m-%d %H:%M:%S')" > $$REPORT_FILE; \
	echo "" >> $$REPORT_FILE; \
	echo "\`\`\`" >> $$REPORT_FILE; \
	benchstat benchstat/previous.txt benchstat/latest.txt >> $$REPORT_FILE; \
	echo "\`\`\`" >> $$REPORT_FILE; \
	echo "" >> $$REPORT_FILE; \
	echo "Report saved to $$REPORT_FILE"

# Run cross-language benchmarks to compare Go and Python implementations
cross-benchmark:
	@echo "Running cross-language benchmarks..."
	@cmd/benchmark/run_benchmarks.sh --output-dir benchstat/cross
	@echo "Cross-language benchmark report saved to benchstat/cross/comparison_report.txt"

# Compare this library with other Golang Jinja template libraries
golang-jinja-compare:
	@echo "Comparing with other Golang Jinja implementations..."
	@# Make sure pongo2 dependencies are properly set up in its own module
	@cd cmd/benchmark/pongo2_benchmark && go mod tidy
	@# Run the comparison benchmark
	@cmd/benchmark/run_benchmarks.sh --output-dir benchstat/golang-compare
	@echo "Comparison report saved to benchstat/golang-compare/comparison_report.txt" 