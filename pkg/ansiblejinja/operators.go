package ansiblejinja

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// isTruthy determines if a value is considered "truthy" in the Python/Jinja2 sense
func isTruthy(value interface{}) bool {
	if value == nil {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v != ""
	case []interface{}:
		return len(v) > 0
	case map[string]interface{}:
		return len(v) > 0
	default:
		// Try reflection for slices and maps
		rv := reflect.ValueOf(value)
		kind := rv.Kind()
		if kind == reflect.Slice || kind == reflect.Array || kind == reflect.Map {
			return rv.Len() > 0
		}
		// For other types, treat as truthy by default
		return true
	}
}

// negateValue negates a numeric value
func negateValue(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case int:
		return -v, nil
	case float64:
		return -v, nil
	default:
		return nil, fmt.Errorf("cannot apply unary minus to non-numeric type: %T", value)
	}
}

// equals checks if two values are equal, with type coercion rules similar to Python
func equals(left, right interface{}) (interface{}, error) {
	// Handle nil/None values
	if left == nil && right == nil {
		return true, nil
	}
	if left == nil || right == nil {
		return false, nil
	}

	// Type-specific equality
	switch l := left.(type) {
	case string:
		if r, ok := right.(string); ok {
			return l == r, nil
		}
	case int:
		switch r := right.(type) {
		case int:
			return l == r, nil
		case float64:
			return float64(l) == r, nil
		}
	case float64:
		switch r := right.(type) {
		case int:
			return l == float64(r), nil
		case float64:
			return l == r, nil
		}
	case bool:
		if r, ok := right.(bool); ok {
			return l == r, nil
		}
	case []interface{}:
		if r, ok := right.([]interface{}); ok {
			// Special case for empty slices - both empty slices should be equal
			if len(l) == 0 && len(r) == 0 {
				return true, nil
			}
			if len(l) != len(r) {
				return false, nil
			}
			for i := range l {
				eq, err := equals(l[i], r[i])
				if err != nil {
					return nil, err
				}
				if !eq.(bool) {
					return false, nil
				}
			}
			return true, nil
		}
	case map[string]interface{}:
		if r, ok := right.(map[string]interface{}); ok {
			if len(l) != len(r) {
				return false, nil
			}
			for k, v := range l {
				rv, ok := r[k]
				if !ok {
					return false, nil
				}
				eq, err := equals(v, rv)
				if err != nil {
					return nil, err
				}
				if !eq.(bool) {
					return false, nil
				}
			}
			return true, nil
		}
	}

	// Default to structural equality using reflect.DeepEqual
	return reflect.DeepEqual(left, right), nil
}

// compare performs a comparison operation between two values
func compare(left, right interface{}, comparator func(float64, float64) bool) (interface{}, error) {
	var leftFloat, rightFloat float64
	var err error

	leftFloat, err = toFloat(left)
	if err != nil {
		return nil, fmt.Errorf("left operand: %v", err)
	}

	rightFloat, err = toFloat(right)
	if err != nil {
		return nil, fmt.Errorf("right operand: %v", err)
	}

	return comparator(leftFloat, rightFloat), nil
}

// toFloat converts a value to a float64 for comparison operations
func toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case int:
		return float64(v), nil
	case float64:
		return v, nil
	case string:
		// Try to parse string as number
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to number", v)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to number", value)
	}
}

// toInteger converts a value to an integer for indexing
func toInteger(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		i, err := strconv.Atoi(v)
		if err != nil {
			return 0, fmt.Errorf("cannot convert string '%s' to integer", v)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to integer", value)
	}
}

// checkMembership checks if an item is in a collection
func checkMembership(collection, item interface{}) (interface{}, error) {
	switch c := collection.(type) {
	case string:
		// Check if a character/substring is in a string
		if s, ok := item.(string); ok {
			return strings.Contains(c, s), nil
		}
		return false, nil

	case []interface{}:
		// Check if an item is in a list
		for _, v := range c {
			eq, err := equals(v, item)
			if err != nil {
				continue
			}
			if eq.(bool) {
				return true, nil
			}
		}
		return false, nil

	case map[string]interface{}:
		// Check if a key is in a dictionary
		key := fmt.Sprintf("%v", item)
		_, ok := c[key]
		return ok, nil

	default:
		// Try reflection for other collection types
		rv := reflect.ValueOf(collection)
		kind := rv.Kind()
		if kind == reflect.Slice || kind == reflect.Array {
			for i := 0; i < rv.Len(); i++ {
				eq, err := equals(rv.Index(i).Interface(), item)
				if err != nil {
					continue
				}
				if eq.(bool) {
					return true, nil
				}
			}
			return false, nil
		} else if kind == reflect.Map {
			for _, key := range rv.MapKeys() {
				eq, err := equals(key.Interface(), item)
				if err != nil {
					continue
				}
				if eq.(bool) {
					return true, nil
				}
			}
			return false, nil
		}

		return false, fmt.Errorf("'in' operator not supported for type: %T", collection)
	}
}

// Mathematical operation functions

// add adds two values, with type coercion similar to Python
func add(left, right interface{}) (interface{}, error) {
	// String concatenation
	if lstr, ok := left.(string); ok {
		if rstr, ok := right.(string); ok {
			return lstr + rstr, nil
		}
		return nil, fmt.Errorf("cannot concatenate string with %T", right)
	}

	// List concatenation
	if llist, ok := left.([]interface{}); ok {
		if rlist, ok := right.([]interface{}); ok {
			result := make([]interface{}, len(llist)+len(rlist))
			copy(result, llist)
			copy(result[len(llist):], rlist)
			return result, nil
		}
		return nil, fmt.Errorf("cannot concatenate list with %T", right)
	}

	// Numeric addition
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot add %T and %T", left, right)
	}

	// If both inputs were integers, return integer
	if _, lok := left.(int); lok {
		if _, rok := right.(int); rok {
			return int(lnum) + int(rnum), nil
		}
	}

	// Otherwise return float
	return lnum + rnum, nil
}

// subtract subtracts two values
func subtract(left, right interface{}) (interface{}, error) {
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot subtract %T from %T", right, left)
	}

	// If both inputs were integers, return integer
	if _, lok := left.(int); lok {
		if _, rok := right.(int); rok {
			return int(lnum) - int(rnum), nil
		}
	}

	// Otherwise return float
	return lnum - rnum, nil
}

// multiply multiplies two values
func multiply(left, right interface{}) (interface{}, error) {
	// String repetition: "a" * 3 = "aaa"
	if lstr, ok := left.(string); ok {
		if rnum, ok := right.(int); ok {
			return strings.Repeat(lstr, rnum), nil
		}
		return nil, fmt.Errorf("cannot multiply string by %T", right)
	}

	// List repetition: [1, 2] * 3 = [1, 2, 1, 2, 1, 2]
	if llist, ok := left.([]interface{}); ok {
		if rnum, ok := right.(int); ok {
			if rnum <= 0 {
				return []interface{}{}, nil
			}
			result := make([]interface{}, 0, len(llist)*rnum)
			for i := 0; i < rnum; i++ {
				result = append(result, llist...)
			}
			return result, nil
		}
		return nil, fmt.Errorf("cannot multiply list by %T", right)
	}

	// Numeric multiplication
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot multiply %T and %T", left, right)
	}

	// If both inputs were integers, return integer
	if _, lok := left.(int); lok {
		if _, rok := right.(int); rok {
			return int(lnum) * int(rnum), nil
		}
	}

	// Otherwise return float
	return lnum * rnum, nil
}

// divide divides two values (true division like Python's /)
func divide(left, right interface{}) (interface{}, error) {
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot divide %T by %T", left, right)
	}

	if rnum == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	// Always return float for true division
	return lnum / rnum, nil
}

// floorDivide performs floor division like Python's //
func floorDivide(left, right interface{}) (interface{}, error) {
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot floor divide %T by %T", left, right)
	}

	if rnum == 0 {
		return nil, fmt.Errorf("division by zero")
	}

	// If both inputs were integers, return integer
	if _, lok := left.(int); lok {
		if _, rok := right.(int); rok {
			return int(math.Floor(lnum / rnum)), nil
		}
	}

	// Otherwise return float
	return math.Floor(lnum / rnum), nil
}

// modulo performs the modulo operation
func modulo(left, right interface{}) (interface{}, error) {
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot compute modulo of %T and %T", left, right)
	}

	if rnum == 0 {
		return nil, fmt.Errorf("modulo by zero")
	}

	// If both inputs were integers, return integer
	if _, lok := left.(int); lok {
		if _, rok := right.(int); rok {
			return int(lnum) % int(rnum), nil
		}
	}

	// Otherwise use fmod for floating point modulo
	return math.Mod(lnum, rnum), nil
}

// power calculates the power of a value (exponentiation)
func power(left, right interface{}) (interface{}, error) {
	lnum, lerr := toFloat(left)
	rnum, rerr := toFloat(right)
	if lerr != nil || rerr != nil {
		return nil, fmt.Errorf("cannot compute power of %T and %T", left, right)
	}

	// If both inputs were integers and the exponent is positive,
	// we can return an integer
	if _, lok := left.(int); lok {
		if rInt, rok := right.(int); rok && rInt >= 0 {
			return int(math.Pow(lnum, rnum)), nil
		}
	}

	// Otherwise return float
	return math.Pow(lnum, rnum), nil
}

// getAttributeValue gets an attribute from an object (obj.attr)
func getAttributeValue(obj interface{}, attr string) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot access attribute '%s' of nil", attr)
	}

	// For maps, just look up the key
	if m, ok := obj.(map[string]interface{}); ok {
		if val, exists := m[attr]; exists {
			return val, nil
		}
		return nil, fmt.Errorf("attribute '%s' not found in map", attr)
	}

	// Use reflection for struct field access
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("cannot access attribute of non-struct, non-map type: %T", obj)
	}

	// First try to find a field with exactly matching name
	field := v.FieldByName(attr)
	if field.IsValid() {
		return field.Interface(), nil
	}

	// Then try for a method with the given name
	method := v.MethodByName(attr)
	if method.IsValid() {
		return method.Interface(), nil
	}

	// Try case-insensitive match for the field
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		if strings.EqualFold(t.Field(i).Name, attr) {
			return v.Field(i).Interface(), nil
		}
	}

	// Try case-insensitive match for the method
	for i := 0; i < t.NumMethod(); i++ {
		if strings.EqualFold(t.Method(i).Name, attr) {
			return v.Method(i).Interface(), nil
		}
	}

	return nil, fmt.Errorf("attribute '%s' not found in %T", attr, obj)
}

// getSubscriptValue gets a value by subscript access (obj[key])
func getSubscriptValue(obj, key interface{}) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot subscript nil value")
	}

	switch o := obj.(type) {
	case map[string]interface{}:
		// For maps, convert key to string and look it up
		k := fmt.Sprintf("%v", key)
		if val, exists := o[k]; exists {
			return val, nil
		}
		return nil, fmt.Errorf("key '%v' not found in map", key)

	case []interface{}:
		// For lists, key must be an integer index
		idx, err := toInteger(key)
		if err != nil {
			return nil, fmt.Errorf("list index must be an integer, got %T", key)
		}

		// Handle negative index (Python-style)
		if idx < 0 {
			idx += len(o)
		}

		if idx < 0 || idx >= len(o) {
			return nil, fmt.Errorf("list index out of range: %d", idx)
		}

		return o[idx], nil

	case string:
		// For strings, key must be an integer index
		idx, err := toInteger(key)
		if err != nil {
			return nil, fmt.Errorf("string index must be an integer, got %T", key)
		}

		// Handle negative index (Python-style)
		if idx < 0 {
			idx += len(o)
		}

		if idx < 0 || idx >= len(o) {
			return nil, fmt.Errorf("string index out of range: %d", idx)
		}

		return string(o[idx]), nil

	default:
		// Try reflection for other types
		v := reflect.ValueOf(obj)
		kind := v.Kind()

		if kind == reflect.Map {
			mapKey := reflect.ValueOf(key)
			if !mapKey.Type().AssignableTo(v.Type().Key()) {
				return nil, fmt.Errorf("key type mismatch: expected %v, got %T", v.Type().Key(), key)
			}

			val := v.MapIndex(mapKey)
			if !val.IsValid() {
				return nil, fmt.Errorf("key '%v' not found in map", key)
			}

			return val.Interface(), nil
		}

		if kind == reflect.Slice || kind == reflect.Array {
			idx, err := toInteger(key)
			if err != nil {
				return nil, fmt.Errorf("index must be an integer, got %T", key)
			}

			// Handle negative index (Python-style)
			if idx < 0 {
				idx += v.Len()
			}

			if idx < 0 || idx >= v.Len() {
				return nil, fmt.Errorf("index out of range: %d", idx)
			}

			return v.Index(idx).Interface(), nil
		}

		return nil, fmt.Errorf("cannot subscript type: %T", obj)
	}
}

// callFunction calls a function with the provided arguments
func callFunction(callable interface{}, args []interface{}) (interface{}, error) {
	// Handle function objects
	fn, ok := callable.(func(...interface{}) (interface{}, error))
	if ok {
		return fn(args...)
	}

	// Use reflection for method calls and other callable types
	v := reflect.ValueOf(callable)
	if v.Kind() != reflect.Func {
		return nil, fmt.Errorf("object is not callable: %T", callable)
	}

	// Convert arguments to reflect values
	reflectArgs := make([]reflect.Value, len(args))
	for i, arg := range args {
		reflectArgs[i] = reflect.ValueOf(arg)
	}

	// Call the function
	result := v.Call(reflectArgs)

	// Handle the return value(s)
	if len(result) == 0 {
		return nil, nil
	} else if len(result) == 1 {
		return result[0].Interface(), nil
	} else {
		// Multiple return values - convert to a slice
		values := make([]interface{}, len(result))
		for i, r := range result {
			values[i] = r.Interface()
		}
		return values, nil
	}
}
