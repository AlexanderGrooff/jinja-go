package jinja

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
)

// FunctionFunc defines the signature for a function callable from templates
type FunctionFunc func(args ...interface{}) (interface{}, error)

// GlobalFunctions stores the registered functions that can be called directly in templates
var GlobalFunctions map[string]FunctionFunc

// GlobalMethods stores methods that can be called on objects of specific types
var GlobalMethods map[string]map[string]FunctionFunc

// Initialize the GlobalFunctions map and register all functions
func init() {
	GlobalFunctions = make(map[string]FunctionFunc)
	GlobalMethods = make(map[string]map[string]FunctionFunc)

	// Register the lookup function
	GlobalFunctions["lookup"] = lookupFunction

	// Register other functions as they are implemented

	// Register methods for map type
	registerMapMethods()

	// Register methods for string type
	registerStringMethods()
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

// registerMapMethods registers methods that can be called on map types
func registerMapMethods() {
	// Create methods map for map type
	mapMethods := make(map[string]FunctionFunc)

	// Register the get method
	mapMethods["get"] = mapGetMethod

	// Add map methods to global methods
	GlobalMethods["map"] = mapMethods
}

// registerStringMethods registers methods that can be called on string types
func registerStringMethods() {
	// Create methods map for string type
	stringMethods := make(map[string]FunctionFunc)

	// Register the format method
	stringMethods["format"] = stringFormatMethod

	// Add string methods to global methods
	GlobalMethods["string"] = stringMethods
}

// mapGetMethod implements the dictionary get method:
// Usage: {{ my_dict.get('key') }} -> returns the value for key
// Usage: {{ my_dict.get('key', 'default') }} -> returns value for key or default if key doesn't exist
func mapGetMethod(args ...interface{}) (interface{}, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("get method requires at least a dictionary and a key")
	}

	// First argument is the dictionary/map itself
	dict := args[0]
	if dict == nil {
		if len(args) > 2 {
			// If dictionary is nil and default is provided, return default
			return args[2], nil
		}
		return nil, nil
	}

	// Second argument is the key
	key := args[1]

	// Try to get the value based on the type of dictionary
	switch d := dict.(type) {
	case map[string]interface{}:
		// Convert key to string for string-keyed maps
		strKey := fmt.Sprintf("%v", key)
		if val, ok := d[strKey]; ok {
			return val, nil
		}

		// Key not found - return default value if provided
		if len(args) > 2 {
			return args[2], nil
		}
		return nil, nil

	case map[interface{}]interface{}:
		// Try direct key access first
		if val, ok := d[key]; ok {
			return val, nil
		}

		// Try string conversion of key
		strKey := fmt.Sprintf("%v", key)
		if val, ok := d[strKey]; ok {
			return val, nil
		}

		// Key not found - return default value if provided
		if len(args) > 2 {
			return args[2], nil
		}
		return nil, nil

	default:
		// For other types, try using reflection
		v := reflect.ValueOf(dict)
		if v.Kind() == reflect.Map {
			keyVal := reflect.ValueOf(key)

			// Check if key is directly usable as map key
			if keyVal.Type().AssignableTo(v.Type().Key()) {
				val := v.MapIndex(keyVal)
				if val.IsValid() {
					return val.Interface(), nil
				}
			}

			// Try with string conversion of key
			strKey := fmt.Sprintf("%v", key)
			strKeyVal := reflect.ValueOf(strKey)
			if strKeyVal.Type().AssignableTo(v.Type().Key()) {
				val := v.MapIndex(strKeyVal)
				if val.IsValid() {
					return val.Interface(), nil
				}
			}

			// Key not found - return default value if provided
			if len(args) > 2 {
				return args[2], nil
			}
			return nil, nil
		}

		return nil, fmt.Errorf("get method requires a dictionary/map, got %T", dict)
	}
}

// stringFormatMethod implements the string format method:
// Usage: {{ "Hello, {}!".format("world") }} -> "Hello, world!"
// Usage: {{ "Hello, {name}!".format(name="world") }} -> "Hello, world!"
// Usage: {{ "{0}, {1}, {2}".format("a", "b", "c") }} -> "a, b, c"
func stringFormatMethod(args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("format method requires a string")
	}

	// First argument is the string itself
	formatStr, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("format method requires a string, got %T", args[0])
	}

	// If no arguments provided other than the string itself, return the string unmodified
	if len(args) == 1 {
		return formatStr, nil
	}

	// All args after the format string itself are format arguments
	formatArgs := args[1:]

	// For auto-numbering - track the next positional arg to use
	nextArg := 0

	// Pattern to match format placeholders: {}, {0}, {name}, etc.
	placeholderPattern := regexp.MustCompile(`{([^{}]*)}`)

	// Replace placeholders with values
	result := placeholderPattern.ReplaceAllStringFunc(formatStr, func(placeholder string) string {
		// Extract the name/index inside braces, removing the braces
		name := placeholder[1 : len(placeholder)-1]

		// Empty placeholder {} - use the next positional argument
		if name == "" {
			if nextArg >= len(formatArgs) {
				// Lacking a positional argument - return the original placeholder
				return placeholder
			}
			// Use the next positional argument
			arg := formatArgs[nextArg]
			nextArg++ // Move to next arg for next auto-numbered placeholder
			return fmt.Sprintf("%v", arg)
		}

		// Numeric placeholder {0}, {1}, etc. - use the indexed positional argument
		if index, err := strconv.Atoi(name); err == nil {
			if index >= 0 && index < len(formatArgs) {
				return fmt.Sprintf("%v", formatArgs[index])
			}
			// Index out of range - return the original placeholder
			return placeholder
		}

		// Named placeholder {name} - check if any arg is a map with this key
		// This is a simplification as Jinja/Python would use keyword arguments
		for _, arg := range formatArgs {
			if m, ok := arg.(map[string]interface{}); ok {
				if val, exists := m[name]; exists {
					return fmt.Sprintf("%v", val)
				}
			}
		}

		// Cannot resolve the placeholder - return as is
		return placeholder
	})

	return result, nil
}
