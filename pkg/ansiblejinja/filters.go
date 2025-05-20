package ansiblejinja

import (
	"fmt"
	"reflect"
	"strings"
)

// FilterFunc defines the signature for a filter function.
// input is the value to be filtered.
// args are the arguments passed to the filter.
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)

// GlobalFilters stores the registered filter functions.
var GlobalFilters = map[string]FilterFunc{
	"default": defaultFilter,
	"join":    joinFilter,
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
