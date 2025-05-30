package jinja

import (
	"fmt"
	"html"
	"os"
	"reflect"
	"strings"
)

// FilterFunc defines the signature for a filter function.
// input is the value to be filtered.
// args are the arguments passed to the filter.
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)

// GlobalFilters stores the registered filter functions.
var GlobalFilters map[string]FilterFunc

// defaultFilter implements the 'default' Jinja filter.
// If the input value is considered "falsy" (nil, false, empty string, empty slice/map),
// it returns the default_value. Otherwise, it returns the input value.
// Numbers (including 0) are not considered falsy by this filter.
func defaultFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("default filter requires at least one argument (the default value)")
	}
	// TODO: The spec allows a second boolean argument to default filter for strict undefined check.
	// e.g., {{ my_var | default("val", true) }} only defaults if my_var is undefined, not if it's just falsy.
	// This is not implemented yet. We are implementing the common one-argument behavior.
	defaultValue := args[0]

	if input == nil {
		return defaultValue, nil
	}

	val := reflect.ValueOf(input)
	switch val.Kind() {
	// Note: reflect.Invalid is not typically expected here if input is not nil,
	// but can occur if input was, for example, a nil interface that wasn't caught by `input == nil`.
	case reflect.Invalid:
		return defaultValue, nil
	case reflect.Bool:
		if !val.Bool() {
			return defaultValue, nil
		}
	case reflect.String:
		if val.Len() == 0 {
			return defaultValue, nil
		}
	case reflect.Slice, reflect.Array, reflect.Map:
		if val.Len() == 0 {
			return defaultValue, nil
		}
		// Numbers (int, float, etc.) including 0 are not considered falsy by the default filter.
		// So, if input is 0, it will be returned as is.
	}

	return input, nil
}

// joinFilter implements the 'join' Jinja filter.
// It joins the elements of a sequence (array, slice) with a given delimiter.
// Usage: {{ ['a', 'b', 'c'] | join(',') }} -> "a,b,c"
func joinFilter(input interface{}, args ...interface{}) (interface{}, error) {
	// Default delimiter is an empty string if not specified
	delimiter := ""
	if len(args) > 0 {
		if delim, ok := args[0].(string); ok {
			delimiter = delim
		} else {
			return nil, fmt.Errorf("join filter delimiter must be a string")
		}
	}

	if input == nil {
		return "", nil
	}

	val := reflect.ValueOf(input)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		length := val.Len()
		elements := make([]string, 0, length)

		for i := 0; i < length; i++ {
			itemVal := val.Index(i).Interface()
			// Convert each element to string
			elements = append(elements, fmt.Sprintf("%v", itemVal))
		}

		return strings.Join(elements, delimiter), nil
	case reflect.String:
		// If the input is already a string, return it unchanged
		return input, nil
	default:
		return nil, fmt.Errorf("join filter requires a sequence (array, slice) as input, got %T", input)
	}
}

// upperFilter implements the 'upper' Jinja filter.
// It converts a string to uppercase.
// Usage: {{ 'Hello' | upper }} -> "HELLO"
func upperFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return "", nil
	}

	switch v := input.(type) {
	case string:
		return strings.ToUpper(v), nil
	default:
		// Try to convert to string
		str := fmt.Sprintf("%v", input)
		return strings.ToUpper(str), nil
	}
}

// lowerFilter implements the 'lower' Jinja filter.
// It converts a string to lowercase.
// Usage: {{ 'Hello' | lower }} -> "hello"
func lowerFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return "", nil
	}

	switch v := input.(type) {
	case string:
		return strings.ToLower(v), nil
	default:
		// Try to convert to string
		str := fmt.Sprintf("%v", input)
		return strings.ToLower(str), nil
	}
}

// capitalizeFilter implements the 'capitalize' Jinja filter.
// It capitalizes the first character of a string and lowercases the rest.
// Usage: {{ 'hello world' | capitalize }} -> "Hello world"
func capitalizeFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return "", nil
	}

	var str string
	switch v := input.(type) {
	case string:
		str = v
	default:
		// Try to convert to string
		str = fmt.Sprintf("%v", input)
	}

	if str == "" {
		return "", nil
	}

	// Capitalize first letter, lowercase the rest
	return strings.ToUpper(str[:1]) + strings.ToLower(str[1:]), nil
}

// replaceFilter implements the 'replace' Jinja filter.
// It replaces occurrences of a substring with another.
// Usage: {{ 'Hello World' | replace('Hello', 'Hi') }} -> "Hi World"
func replaceFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return "", nil
	}

	if len(args) < 2 {
		return nil, fmt.Errorf("replace filter requires two arguments: old substring and new substring")
	}

	old, ok1 := args[0].(string)
	new, ok2 := args[1].(string)

	if !ok1 || !ok2 {
		return nil, fmt.Errorf("replace filter arguments must be strings")
	}

	// If args[2] exists and is an int, it's the count of replacements
	count := -1 // default: replace all
	if len(args) > 2 {
		if countVal, ok := args[2].(int); ok {
			count = countVal
		}
	}

	var str string
	switch v := input.(type) {
	case string:
		str = v
	default:
		// Try to convert to string
		str = fmt.Sprintf("%v", input)
	}

	return strings.Replace(str, old, new, count), nil
}

// trimFilter implements the 'trim' Jinja filter.
// It removes leading and trailing whitespace or specified characters.
// Usage: {{ '  Hello  ' | trim }} -> "Hello"
// Usage: {{ 'Hello World' | trim('Hld') }} -> "ello Wor"
func trimFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return "", nil
	}

	var str string
	switch v := input.(type) {
	case string:
		str = v
	default:
		// Try to convert to string
		str = fmt.Sprintf("%v", input)
	}

	// If no cutset is provided, trim whitespace
	if len(args) == 0 {
		return strings.TrimSpace(str), nil
	}

	// If cutset is provided, use it
	cutset, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("trim filter cutset argument must be a string")
	}

	return strings.Trim(str, cutset), nil
}

// listFilter implements the 'list' Jinja filter.
// It converts a value to a list. If the input is a string, it returns a list of characters.
// Usage: {{ 'abc' | list }} -> ['a', 'b', 'c']
func listFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return []interface{}{}, nil
	}

	val := reflect.ValueOf(input)

	switch val.Kind() {
	case reflect.String:
		str := val.String()
		result := make([]interface{}, 0, len(str))
		for _, ch := range str {
			result = append(result, string(ch))
		}
		return result, nil
	case reflect.Slice, reflect.Array:
		// If already a slice or array, return a copy to ensure it's []interface{}
		length := val.Len()
		result := make([]interface{}, length)
		for i := 0; i < length; i++ {
			result[i] = val.Index(i).Interface()
		}
		return result, nil
	default:
		// For other types, return a single-item list containing the input
		return []interface{}{input}, nil
	}
}

// escapeFilter implements the 'escape' Jinja filter.
// It escapes special characters in HTML (&, <, >, ", ').
// Usage: {{ '<div>' | escape }} -> "&lt;div&gt;"
func escapeFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return "", nil
	}

	var str string
	switch v := input.(type) {
	case string:
		str = v
	default:
		// Try to convert to string
		str = fmt.Sprintf("%v", input)
	}

	return html.EscapeString(str), nil
}

// mapFilter implements the 'map' Jinja filter.
// It applies a filter to each item in a sequence and returns a list of results.
// Usage: {{ [1, 2, 3] | map('upper') }} -> ["1", "2", "3"]
// Usage: {{ ['a', 'b'] | map('upper') }} -> ["A", "B"]
func mapFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("map filter requires at least one argument (the filter name)")
	}

	// Get the filter name from the first argument
	filterName, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("map filter first argument must be a string (filter name)")
	}

	// Look up the filter function
	filterFunc, exists := GlobalFilters[filterName]
	if !exists {
		return nil, fmt.Errorf("filter '%s' not found", filterName)
	}

	// Additional arguments to pass to the filter function
	filterArgs := args[1:]

	// Check if input is nil
	if input == nil {
		return []interface{}{}, nil
	}

	val := reflect.ValueOf(input)
	switch val.Kind() {
	case reflect.Slice, reflect.Array:
		length := val.Len()
		result := make([]interface{}, 0, length)

		for i := 0; i < length; i++ {
			itemVal := val.Index(i).Interface()
			// Apply the filter to each item
			filteredItem, err := filterFunc(itemVal, filterArgs...)
			if err != nil {
				return nil, fmt.Errorf("error applying filter '%s' to item: %v", filterName, err)
			}
			result = append(result, filteredItem)
		}

		return result, nil
	default:
		// For non-sequence types, apply the filter to the input directly
		return filterFunc(input, filterArgs...)
	}
}

// itemsFilter implements the 'items' Jinja filter.
// It converts a dictionary/map into a list of key-value pairs.
// Usage: {{ {'a': 1, 'b': 2} | items }} -> [('a', 1), ('b', 2)]
func itemsFilter(input interface{}, args ...interface{}) (interface{}, error) {
	if input == nil {
		return []interface{}{}, nil
	}

	val := reflect.ValueOf(input)

	// Only process map types
	if val.Kind() != reflect.Map {
		return nil, fmt.Errorf("items filter requires a dictionary/map as input, got %T", input)
	}

	// Get all keys from the map
	keys := val.MapKeys()
	result := make([]interface{}, 0, len(keys))

	// For each key, create a tuple (key, value) and add to result
	for _, key := range keys {
		value := val.MapIndex(key)
		pair := []interface{}{key.Interface(), value.Interface()}
		result = append(result, pair)
	}

	return result, nil
}

// lookupFilter implements the 'lookup' Ansible filter.
// It retrieves data from external sources based on lookup type.
// Usage: {{ lookup('file', '/path/to/file') }}
// Usage: {{ lookup('env', 'HOME') }}
func lookupFilter(input interface{}, args ...interface{}) (interface{}, error) {
	// For the lookup filter, the input is actually the first argument (lookup type)
	// and the remaining args are passed to the specific lookup function
	if input == nil {
		return nil, fmt.Errorf("lookup filter requires a lookup type as input")
	}

	// Convert input to string
	lookupType, ok := input.(string)
	if !ok {
		return nil, fmt.Errorf("lookup filter requires a string as lookup type, got %T", input)
	}

	switch lookupType {
	case "file":
		if len(args) < 1 {
			return nil, fmt.Errorf("file lookup requires a file path argument")
		}
		filePath, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("file lookup requires a string as file path, got %T", args[0])
		}

		// Read the file content
		content, err := readFileContent(filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading file '%s': %v", filePath, err)
		}
		return content, nil

	case "env":
		if len(args) < 1 {
			return nil, fmt.Errorf("env lookup requires an environment variable name")
		}
		envVar, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("env lookup requires a string as environment variable name, got %T", args[0])
		}

		// Get the environment variable
		value := os.Getenv(envVar)
		return value, nil

	// Add more lookup types as needed

	default:
		return nil, fmt.Errorf("unsupported lookup type: %s", lookupType)
	}
}

// Helper function to read file content
func readFileContent(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func init() {
	// Initialize GlobalFilters after all filter functions are defined
	GlobalFilters = map[string]FilterFunc{
		"default":    defaultFilter,
		"join":       joinFilter,
		"upper":      upperFilter,
		"lower":      lowerFilter,
		"capitalize": capitalizeFilter,
		"replace":    replaceFilter,
		"trim":       trimFilter,
		"list":       listFilter,
		"escape":     escapeFilter,
		"map":        mapFilter,
		"items":      itemsFilter,
		"lookup":     lookupFilter,
	}
}
