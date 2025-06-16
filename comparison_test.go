package jinja

import (
	"testing"
)

func TestComparisons(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		context  map[string]interface{}
		expected interface{}
	}{
		// Integer comparisons
		{
			name:     "int greater than",
			expr:     "a > b",
			context:  map[string]interface{}{"a": 40, "b": 0},
			expected: true,
		},
		{
			name:     "int equal",
			expr:     "a == b",
			context:  map[string]interface{}{"a": 42, "b": 42},
			expected: true,
		},
		{
			name:     "int not equal",
			expr:     "a != b",
			context:  map[string]interface{}{"a": 40, "b": 0},
			expected: true,
		},

		// String comparisons
		{
			name:     "string less than",
			expr:     "a < b",
			context:  map[string]interface{}{"a": "apple", "b": "banana"},
			expected: true,
		},
		{
			name:     "string equal",
			expr:     "a == b",
			context:  map[string]interface{}{"a": "hello", "b": "hello"},
			expected: true,
		},

		// Mixed type comparisons
		{
			name:     "int vs float",
			expr:     "a == b",
			context:  map[string]interface{}{"a": 42, "b": 42.0},
			expected: true,
		},
		{
			name:     "int vs float greater",
			expr:     "a > b",
			context:  map[string]interface{}{"a": 43, "b": 42.5},
			expected: true,
		},

		// Boolean comparisons
		{
			name:     "bool equal",
			expr:     "a == b",
			context:  map[string]interface{}{"a": true, "b": true},
			expected: true,
		},
		{
			name:     "bool less than",
			expr:     "a < b",
			context:  map[string]interface{}{"a": false, "b": true},
			expected: true,
		},

		// Nested attribute comparison (like the original bug)
		{
			name: "nested attribute comparison",
			expr: "stat_result.stat.size > 0",
			context: map[string]interface{}{
				"stat_result": map[string]interface{}{
					"stat": map[string]interface{}{
						"size": 40,
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EvaluateExpression(tt.expr, tt.context)
			if err != nil {
				t.Fatalf("EvaluateExpression failed: %v", err)
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
