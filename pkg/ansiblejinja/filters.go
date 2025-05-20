package ansiblejinja

import (
	"fmt"
	"html"
	"reflect"
	"strings"
)

// FilterFunc defines the signature for a filter function.
// input is the value to be filtered.
// args are the arguments passed to the filter.
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)

// GlobalFilters stores the registered filter functions.
var GlobalFilters = map[string]FilterFunc{
	"default":    defaultFilter,
	"join":       joinFilter,
	"upper":      upperFilter,
	"lower":      lowerFilter,
	"capitalize": capitalizeFilter,
	"replace":    replaceFilter,
	"trim":       trimFilter,
	"list":       listFilter,
	"escape":     escapeFilter,
}

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
