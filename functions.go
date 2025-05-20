package ansiblejinja

import (
	"fmt"
	"os"
)

// FunctionFunc defines the signature for a function callable from templates
type FunctionFunc func(args ...interface{}) (interface{}, error)

// GlobalFunctions stores the registered functions that can be called directly in templates
var GlobalFunctions map[string]FunctionFunc

// Initialize the GlobalFunctions map and register all functions
func init() {
	GlobalFunctions = make(map[string]FunctionFunc)

	// Register the lookup function
	GlobalFunctions["lookup"] = lookupFunction

	// Register other functions as they are implemented
}

// lookupFunction implements the Ansible 'lookup' function
// Usage: {{ lookup("file", "/path/to/file") }}
// Usage: {{ lookup("env", "HOME") }}
func lookupFunction(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("lookup function requires a lookup type as first argument")
	}

	// Get the lookup type
	lookupType, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("lookup function requires a string as lookup type, got %T", args[0])
	}

	if len(args) < 2 {
		return nil, fmt.Errorf("lookup function requires a target as second argument")
	}

	switch lookupType {
	case "file":
		// File lookup: lookup('file', '/path/to/file')
		filePath, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("file lookup requires a string path, got %T", args[1])
		}

		content, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %v", filePath, err)
		}
		return string(content), nil

	case "env":
		// Environment variable lookup: lookup('env', 'HOME')
		envName, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("env lookup requires a string environment variable name, got %T", args[1])
		}

		// Get environment variable
		envValue := os.Getenv(envName)
		return envValue, nil

	default:
		return nil, fmt.Errorf("unsupported lookup type: %s", lookupType)
	}
}
