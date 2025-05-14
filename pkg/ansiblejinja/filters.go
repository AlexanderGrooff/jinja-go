package ansiblejinja

import (
	"fmt"
	"reflect"
)

// FilterFunc defines the signature for a filter function.
// input is the value to be filtered.
// args are the arguments passed to the filter.
type FilterFunc func(input interface{}, args ...interface{}) (interface{}, error)

// GlobalFilters stores the registered filter functions.
var GlobalFilters = map[string]FilterFunc{
	"default": defaultFilter,
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
