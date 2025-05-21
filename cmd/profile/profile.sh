#!/bin/bash

set -e

# Define output directory
OUTPUT_DIR="profile_results"
TEMPLATES_JSON="cmd/benchmark/templates.json"
ITERATIONS=10000

# Function to display usage instructions
usage() {
    echo "Usage: $0 [options]"
    echo "Options:"
    echo "  --template NAME    Profile a specific template (by name from templates.json)"
    echo "  --iterations N     Number of iterations to run (default: $ITERATIONS)"
    echo "  --cpu              Enable CPU profiling"
    echo "  --mem              Enable memory profiling"
    echo "  --block            Enable block profiling"
    echo "  --all              Enable all profiling types"
    echo "  --all-templates    Run profiling for all templates in templates.json"
    echo "  --help             Show this help message"
}

# Parse command line arguments
TEMPLATE_NAME=""
DO_CPU=false
DO_MEM=false
DO_BLOCK=false
ALL_TEMPLATES=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --template)
            TEMPLATE_NAME="$2"
            shift 2
            ;;
        --iterations)
            ITERATIONS="$2"
            shift 2
            ;;
        --cpu)
            DO_CPU=true
            shift
            ;;
        --mem)
            DO_MEM=true
            shift
            ;;
        --block)
            DO_BLOCK=true
            shift
            ;;
        --all)
            DO_CPU=true
            DO_MEM=true
            DO_BLOCK=true
            shift
            ;;
        --all-templates)
            ALL_TEMPLATES=true
            shift
            ;;
        --help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# If no profiling type is selected, default to CPU
if [ "$DO_CPU" = false ] && [ "$DO_MEM" = false ] && [ "$DO_BLOCK" = false ]; then
    DO_CPU=true
fi

# Build the profiler
echo "Building profiler..."
go build -o profile_tool cmd/profile/main.go

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Function to run profiling for a specific template
run_profile() {
    local name=$1
    local template=$2
    local context=$3
    
    echo "Profiling template: $name"
    
    # Create template directory
    local template_dir="$OUTPUT_DIR/$name"
    mkdir -p "$template_dir"
    
    # Save template and context to files
    echo "$template" > "$template_dir/template.txt"
    echo "$context" > "$template_dir/context.json"
    
    # Build command
    local cmd="./profile_tool --template-string=\"$template\" --context=\"$template_dir/context.json\" --iterations=$ITERATIONS --output-dir=\"$template_dir\""
    
    if [ "$DO_CPU" = true ]; then
        cmd="$cmd --cpuprofile=\"cpu.prof\""
    fi
    
    if [ "$DO_MEM" = true ]; then
        cmd="$cmd --memprofile=\"mem.prof\""
    fi
    
    if [ "$DO_BLOCK" = true ]; then
        cmd="$cmd --blockprofile=\"block.prof\""
    fi
    
    # Run the profiler
    eval "$cmd" | tee "$template_dir/profile_output.txt"
    
    echo "Profile results saved to $template_dir/"
    echo ""
}

# Load templates from benchmark file
TEMPLATES=$(cat "$TEMPLATES_JSON")

if [ "$ALL_TEMPLATES" = true ]; then
    # Run profiling for all templates
    echo "Running profiling for all templates..."
    
    # Extract template names using jq or Python
    if command -v jq >/dev/null 2>&1; then
        TEMPLATE_NAMES=$(echo "$TEMPLATES" | jq -r '.[].name')
    else
        # Fallback to Python if jq is not available
        TEMPLATE_NAMES=$(python3 -c "import json, sys; print('\n'.join(item['name'] for item in json.load(sys.stdin)))" <<< "$TEMPLATES")
    fi
    
    for name in $TEMPLATE_NAMES; do
        # Extract template and context using jq or Python
        if command -v jq >/dev/null 2>&1; then
            TEMPLATE=$(echo "$TEMPLATES" | jq -r ".[] | select(.name == \"$name\") | .template")
            CONTEXT=$(echo "$TEMPLATES" | jq -r ".[] | select(.name == \"$name\") | .context")
        else
            # Fallback to Python if jq is not available
            TEMPLATE=$(python3 -c "import json, sys; data = json.load(sys.stdin); print(next(item['template'] for item in data if item['name'] == '$name'))" <<< "$TEMPLATES")
            CONTEXT=$(python3 -c "import json, sys; data = json.load(sys.stdin); print(json.dumps(next(item['context'] for item in data if item['name'] == '$name')))" <<< "$TEMPLATES")
        fi
        
        run_profile "$name" "$TEMPLATE" "$CONTEXT"
    done
elif [ -n "$TEMPLATE_NAME" ]; then
    # Run profiling for specific template
    echo "Running profiling for template: $TEMPLATE_NAME"
    
    # Extract template and context using jq or Python
    if command -v jq >/dev/null 2>&1; then
        TEMPLATE=$(echo "$TEMPLATES" | jq -r ".[] | select(.name == \"$TEMPLATE_NAME\") | .template")
        CONTEXT=$(echo "$TEMPLATES" | jq -r ".[] | select(.name == \"$TEMPLATE_NAME\") | .context")
    else
        # Fallback to Python if jq is not available
        TEMPLATE=$(python3 -c "import json, sys; data = json.load(sys.stdin); print(next(item['template'] for item in data if item['name'] == '$TEMPLATE_NAME'))" <<< "$TEMPLATES")
        CONTEXT=$(python3 -c "import json, sys; data = json.load(sys.stdin); print(json.dumps(next(item['context'] for item in data if item['name'] == '$TEMPLATE_NAME')))" <<< "$TEMPLATES")
    fi
    
    run_profile "$TEMPLATE_NAME" "$TEMPLATE" "$CONTEXT"
else
    echo "Error: No template specified. Use --template or --all-templates"
    usage
    exit 1
fi

echo "Profiling complete. Use 'go tool pprof' to analyze the results."
echo "Example:"
echo "  go tool pprof -http=:8080 $OUTPUT_DIR/TEMPLATE_NAME/cpu.prof"
echo ""
echo "For text-based analysis:"
echo "  go tool pprof $OUTPUT_DIR/TEMPLATE_NAME/cpu.prof"
echo "  (pprof) top10" 