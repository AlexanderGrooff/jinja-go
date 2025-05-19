package ansiblejinja

import (
	"reflect"
	"testing"
)

func TestParseAndEvaluate(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		context   map[string]interface{}
		want      interface{}
		wantError bool
	}{
		// Basic literals
		{
			name:    "integer literal",
			expr:    "42",
			context: map[string]interface{}{},
			want:    42,
		},
		{
			name:    "float literal",
			expr:    "3.14",
			context: map[string]interface{}{},
			want:    3.14,
		},
		{
			name:    "string literal - single quotes",
			expr:    "'hello'",
			context: map[string]interface{}{},
			want:    "hello",
		},
		{
			name:    "string literal - double quotes",
			expr:    "\"world\"",
			context: map[string]interface{}{},
			want:    "world",
		},
		{
			name:    "boolean literal - true",
			expr:    "True",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "boolean literal - false",
			expr:    "False",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "none literal",
			expr:    "None",
			context: map[string]interface{}{},
			want:    nil,
		},

		// Variable access
		{
			name:    "simple variable",
			expr:    "name",
			context: map[string]interface{}{"name": "Alice"},
			want:    "Alice",
		},
		{
			name:      "undefined variable",
			expr:      "undefined_var",
			context:   map[string]interface{}{},
			wantError: true,
		},

		// Unary operators
		{
			name:    "unary not with true",
			expr:    "not True",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "unary not with false",
			expr:    "not False",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "unary minus with integer",
			expr:    "-5",
			context: map[string]interface{}{},
			want:    -5,
		},
		{
			name:    "unary plus with integer",
			expr:    "+5",
			context: map[string]interface{}{},
			want:    5,
		},

		// Binary operators
		{
			name:    "addition of integers",
			expr:    "2 + 3",
			context: map[string]interface{}{},
			want:    5,
		},
		{
			name:    "subtraction of integers",
			expr:    "5 - 2",
			context: map[string]interface{}{},
			want:    3,
		},
		{
			name:    "multiplication of integers",
			expr:    "3 * 4",
			context: map[string]interface{}{},
			want:    12,
		},
		{
			name:    "division of integers (true division)",
			expr:    "7 / 2",
			context: map[string]interface{}{},
			want:    3.5,
		},
		{
			name:    "floor division of integers",
			expr:    "7 // 2",
			context: map[string]interface{}{},
			want:    3,
		},
		{
			name:    "modulo of integers",
			expr:    "7 % 3",
			context: map[string]interface{}{},
			want:    1,
		},
		{
			name:    "power of integers",
			expr:    "2 ** 3",
			context: map[string]interface{}{},
			want:    8,
		},

		// String operations
		{
			name:    "string concatenation",
			expr:    "'hello' + ' ' + 'world'",
			context: map[string]interface{}{},
			want:    "hello world",
		},
		{
			name:    "string repetition",
			expr:    "'abc' * 3",
			context: map[string]interface{}{},
			want:    "abcabcabc",
		},

		// Comparison operators
		{
			name:    "equals - true case",
			expr:    "1 == 1",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "equals - false case",
			expr:    "1 == 2",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "not equals - true case",
			expr:    "1 != 2",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "not equals - false case",
			expr:    "1 != 1",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "greater than - true case",
			expr:    "5 > 3",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "greater than - false case",
			expr:    "3 > 5",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "less than - true case",
			expr:    "3 < 5",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "less than - false case",
			expr:    "5 < 3",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "greater than or equal - true case (equal)",
			expr:    "5 >= 5",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "greater than or equal - true case (greater)",
			expr:    "5 >= 3",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "greater than or equal - false case",
			expr:    "3 >= 5",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "less than or equal - true case (equal)",
			expr:    "5 <= 5",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "less than or equal - true case (less)",
			expr:    "3 <= 5",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "less than or equal - false case",
			expr:    "5 <= 3",
			context: map[string]interface{}{},
			want:    false,
		},

		// Logical operators with short circuit
		{
			name:    "logical and - both true",
			expr:    "True and True",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "logical and - false and true",
			expr:    "False and True",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "logical and - true and false",
			expr:    "True and False",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "logical and - both false",
			expr:    "False and False",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "logical or - both true",
			expr:    "True or True",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "logical or - false and true",
			expr:    "False or True",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "logical or - true and false",
			expr:    "True or False",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "logical or - both false",
			expr:    "False or False",
			context: map[string]interface{}{},
			want:    false,
		},

		// Identity operators
		{
			name:    "is - true case",
			expr:    "None is None",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "is - false case",
			expr:    "'a' is 'b'",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "is not - true case",
			expr:    "'a' is not 'b'",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "is not - false case",
			expr:    "None is not None",
			context: map[string]interface{}{},
			want:    false,
		},

		// Membership operators
		{
			name:    "in - string contains substring",
			expr:    "'b' in 'abc'",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "in - string doesn't contain substring",
			expr:    "'z' in 'abc'",
			context: map[string]interface{}{},
			want:    false,
		},

		// Complex data types
		{
			name:    "list literal",
			expr:    "[1, 2, 3]",
			context: map[string]interface{}{},
			want:    []interface{}{1, 2, 3},
		},
		{
			name:    "empty list",
			expr:    "[]",
			context: map[string]interface{}{},
			want:    []interface{}{},
		},
		{
			name:    "list with mixed types",
			expr:    "[1, 'two', True]",
			context: map[string]interface{}{},
			want:    []interface{}{1, "two", true},
		},
		{
			name:    "dictionary literal",
			expr:    "{'a': 1, 'b': 2}",
			context: map[string]interface{}{},
			want:    map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:    "empty dictionary",
			expr:    "{}",
			context: map[string]interface{}{},
			want:    map[string]interface{}{},
		},
		{
			name:    "dictionary with mixed value types",
			expr:    "{'a': 1, 'b': 'two', 'c': True}",
			context: map[string]interface{}{},
			want:    map[string]interface{}{"a": 1, "b": "two", "c": true},
		},

		// Attribute access
		{
			name:    "attribute access from map",
			expr:    "user.name",
			context: map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}},
			want:    "Alice",
		},

		// Subscript access
		{
			name:    "subscript access for list",
			expr:    "items[1]",
			context: map[string]interface{}{"items": []interface{}{10, 20, 30}},
			want:    20,
		},
		{
			name:    "subscript access for dictionary",
			expr:    "user['name']",
			context: map[string]interface{}{"user": map[string]interface{}{"name": "Alice"}},
			want:    "Alice",
		},
		{
			name:    "negative index for list",
			expr:    "items[-1]",
			context: map[string]interface{}{"items": []interface{}{10, 20, 30}},
			want:    30,
		},

		// Function calls
		{
			name: "simple function call",
			expr: "add(1, 2)",
			context: map[string]interface{}{
				"add": func(a, b int) int { return a + b },
			},
			want: 3,
		},

		// Precedence
		{
			name:    "operator precedence - multiplication before addition",
			expr:    "1 + 2 * 3",
			context: map[string]interface{}{},
			want:    7,
		},
		{
			name:    "operator precedence - parentheses override",
			expr:    "(1 + 2) * 3",
			context: map[string]interface{}{},
			want:    9,
		},
		{
			name:    "operator precedence - multiple operations",
			expr:    "2 ** 3 * 2 + 3",
			context: map[string]interface{}{},
			want:    19, // (2^3) * 2 + 3 = 8 * 2 + 3 = 16 + 3 = 19
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAndEvaluate(tt.expr, tt.context)

			if (err != nil) != tt.wantError {
				t.Errorf("ParseAndEvaluate() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			// Special handling for empty lists
			if tt.name == "empty list" {
				gotList, gotOk := got.([]interface{})
				wantList, wantOk := tt.want.([]interface{})

				if !gotOk || !wantOk {
					t.Errorf("Expected both got and want to be []interface{}, got: %T, want: %T", got, tt.want)
				} else if len(gotList) != len(wantList) {
					t.Errorf("Expected lists of same length, got: %d, want: %d", len(gotList), len(wantList))
				}
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseAndEvaluate() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}
