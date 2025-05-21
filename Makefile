.PHONY: test benchmark benchmark-save benchmark-compare benchmark-report cross-benchmark profile profile-complex profile-all

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

# Run profiling for a complex template
profile-complex:
	@echo "Running profiling for complex_template..."
	@chmod +x cmd/profile/profile.sh
	@cmd/profile/profile.sh --template complex_template --all

# Run profiling for nested_loops template
profile-nested-loops:
	@echo "Running profiling for nested_loops..."
	@chmod +x cmd/profile/profile.sh
	@cmd/profile/profile.sh --template nested_loops --all

# Run profiling for all templates
profile-all:
	@echo "Running profiling for all templates..."
	@chmod +x cmd/profile/profile.sh
	@cmd/profile/profile.sh --all-templates --all

# Run profiling with custom options
profile:
	@echo "Running custom profiling..."
	@chmod +x cmd/profile/profile.sh
	@cmd/profile/profile.sh $(ARGS) 