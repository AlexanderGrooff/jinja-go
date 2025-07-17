package jinja

import (
	"testing"
)

func TestBasicArithmetic(t *testing.T) {
	tests := []struct {
		name        string
		expression  string
		context     map[string]interface{}
		expected    interface{}
		expectError bool
	}{
		{
			name:       "Simple addition",
			expression: "1 + 1",
			context:    map[string]interface{}{},
			expected:   2,
		},
		{
			name:       "Float addition",
			expression: "1.5 + 2.5",
			context:    map[string]interface{}{},
			expected:   4.0,
		},
		{
			name:       "Integer and float addition",
			expression: "1 + 2.5",
			context:    map[string]interface{}{},
			expected:   3.5,
		},
		{
			name:       "String concatenation",
			expression: "'hello' + ' world'",
			context:    map[string]interface{}{},
			expected:   "hello world",
		},
		{
			name:       "List concatenation",
			expression: "[1, 2] + ['abc']",
			context:    map[string]interface{}{},
			expected:   []interface{}{1, 2, "abc"},
		},
		{
			name:       "Complex list concatenation",
			expression: "[[1] + [2]] + [3, 4]",
			context:    map[string]interface{}{},
			expected:   []interface{}{[]interface{}{1, 2}, 3, 4},
		},
		{
			name:       "Subtraction",
			expression: "10 - 3",
			context:    map[string]interface{}{},
			expected:   7,
		},
		{
			name:       "Multiplication",
			expression: "3 * 4",
			context:    map[string]interface{}{},
			expected:   12,
		},
		{
			name:       "String repetition",
			expression: "'abc' * 3",
			context:    map[string]interface{}{},
			expected:   "abcabcabc",
		},
		{
			name:       "List repetition",
			expression: "[1, 2] * 3",
			context:    map[string]interface{}{},
			expected:   []interface{}{1, 2, 1, 2, 1, 2},
		},
		{
			name:       "Division",
			expression: "10 / 2",
			context:    map[string]interface{}{},
			expected:   5.0,
		},
		{
			name:       "Floor division",
			expression: "10 // 3",
			context:    map[string]interface{}{},
			expected:   3,
		},
		{
			name:       "Modulo",
			expression: "10 % 3",
			context:    map[string]interface{}{},
			expected:   1,
		},
		{
			name:       "Exponentiation",
			expression: "2 ** 3",
			context:    map[string]interface{}{},
			expected:   8,
		},
		{
			name:       "Variable arithmetic",
			expression: "x + y",
			context:    map[string]interface{}{"x": 5, "y": 3},
			expected:   8,
		},
		{
			name:       "Variable list concatenation",
			expression: "list1 + list2",
			context:    map[string]interface{}{"list1": []interface{}{1, 2}, "list2": []interface{}{"a", "b"}},
			expected:   []interface{}{1, 2, "a", "b"},
		},
		{
			name:       "Nested operations",
			expression: "(1 + 2) * 3",
			context:    map[string]interface{}{},
			expected:   9,
		},
		{
			name:       "Complex nested list operations",
			expression: "([1] + [2, 3]) + ([4] + [5])",
			context:    map[string]interface{}{},
			expected:   []interface{}{1, 2, 3, 4, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expression, tt.context)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !deepEqual(result, tt.expected) {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestComplexArithmeticExpressions(t *testing.T) {
	// Test a more complex expression similar to the user's example
	context := map[string]interface{}{
		"interfaces":       []interface{}{"eth0", "eth1"},
		"extra_interfaces": []interface{}{"lo0"},
	}

	// Simpler version of the complex expression for testing
	result, err := EvaluateExpression("interfaces + extra_interfaces", context)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	expected := []interface{}{"eth0", "eth1", "lo0"}
	if !deepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// Helper function for deep comparison
func deepEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch va := a.(type) {
	case []interface{}:
		vb, ok := b.([]interface{})
		if !ok || len(va) != len(vb) {
			return false
		}
		for i := range va {
			if !deepEqual(va[i], vb[i]) {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
