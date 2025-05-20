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
			name:    "string literal - nested quotes",
			expr:    "\"'hello world'\"",
			context: map[string]interface{}{},
			want:    "'hello world'",
		},
		{
			name:    "boolean literal - true uppercase",
			expr:    "True",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "boolean literal - false uppercase",
			expr:    "False",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "boolean literal - true lowercase",
			expr:    "true",
			context: map[string]interface{}{},
			want:    true,
		},
		{
			name:    "boolean literal - false lowercase",
			expr:    "false",
			context: map[string]interface{}{},
			want:    false,
		},
		{
			name:    "none literal - uppercase",
			expr:    "None",
			context: map[string]interface{}{},
			want:    nil,
		},
		{
			name:    "none literal - lowercase",
			expr:    "none",
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
			name:    "addition of floats",
			expr:    "2.5 + 3.5",
			context: map[string]interface{}{},
			want:    6.0,
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

		// Nested dictionary tests
		{
			name:    "nested dictionary literal creation",
			expr:    "{'a': {'b': 1, 'c': 2}, 'd': {'e': 3}}",
			context: map[string]interface{}{},
			want:    map[string]interface{}{"a": map[string]interface{}{"b": 1, "c": 2}, "d": map[string]interface{}{"e": 3}},
		},
		{
			name: "nested dictionary access via subscript",
			expr: "data['a']['b']",
			context: map[string]interface{}{
				"data": map[string]interface{}{
					"a": map[string]interface{}{
						"b": 1,
						"c": 2,
					},
				},
			},
			want: 1,
		},
		{
			name: "nested dictionary access via attributes",
			expr: "data.a.b",
			context: map[string]interface{}{
				"data": map[string]interface{}{
					"a": map[string]interface{}{
						"b": 1,
						"c": 2,
					},
				},
			},
			want: 1,
		},
		{
			name: "mixed access methods - attribute and subscript",
			expr: "data.a['c']",
			context: map[string]interface{}{
				"data": map[string]interface{}{
					"a": map[string]interface{}{
						"b": 1,
						"c": 2,
					},
				},
			},
			want: 2,
		},

		// Function calls
		{
			name:    "get item from dict",
			expr:    "mapping.get('key')",
			context: map[string]interface{}{"mapping": map[string]interface{}{"key": "value"}},
			want:    "value",
		},
		{
			name:    "get item from dict with default - key exists",
			expr:    "mapping.get('key', 'default')",
			context: map[string]interface{}{"mapping": map[string]interface{}{"key": "value"}},
			want:    "value",
		},
		{
			name:    "get item from dict with default - key missing",
			expr:    "mapping.get('missing', 'default')",
			context: map[string]interface{}{"mapping": map[string]interface{}{}},
			want:    "default",
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

// TestEvaluateCompoundExpression specifically tests complex nested expressions
// that require multiple subscript operations
func TestEvaluateCompoundExpression(t *testing.T) {
	tests := []struct {
		name      string
		expr      string
		context   map[string]interface{}
		want      interface{}
		wantError bool
	}{
		{
			name: "deeply nested dictionary",
			expr: "nested_dict['level1']['level2']['level3']['value']",
			context: map[string]interface{}{
				"nested_dict": map[string]interface{}{
					"level1": map[string]interface{}{
						"level2": map[string]interface{}{
							"level3": map[string]interface{}{
								"value": 42,
							},
						},
					},
				},
			},
			want: 42,
		},
		{
			name: "nested dictionary with list values",
			expr: "user_list['users'][1]['name']",
			context: map[string]interface{}{
				"user_list": map[string]interface{}{
					"users": []interface{}{
						map[string]interface{}{
							"name": "Alice",
							"age":  30,
						},
						map[string]interface{}{
							"name": "Bob",
							"age":  25,
						},
					},
				},
			},
			want: "Bob",
		},
		{
			name: "mixed access with variable context",
			expr: "data['items'][0]['name']",
			context: map[string]interface{}{
				"data": map[string]interface{}{
					"items": []interface{}{
						map[string]interface{}{
							"name":  "Item 1",
							"price": 19.99,
						},
						map[string]interface{}{
							"name":  "Item 2",
							"price": 29.99,
						},
					},
				},
			},
			want: "Item 1",
		},
		{
			name: "dictionary in dictionary with mixed keys",
			expr: "nested_map['a']['b']['c']",
			context: map[string]interface{}{
				"nested_map": map[string]interface{}{
					"a": map[string]interface{}{
						"b": map[string]interface{}{
							"c": 42,
						},
					},
				},
			},
			want: 42,
		},
		{
			name: "list in dictionary in list",
			expr: "list_dict_list[0]['items'][1]",
			context: map[string]interface{}{
				"list_dict_list": []interface{}{
					map[string]interface{}{
						"items": []interface{}{10, 20, 30},
					},
				},
			},
			want: 20,
		},
		{
			name:    "evaluate dictionary literal",
			expr:    "{'a': 1, 'b': 2}",
			context: map[string]interface{}{},
			want:    map[string]interface{}{"a": 1, "b": 2},
		},
		{
			name:    "evaluate nested dictionary literal",
			expr:    "{'a': {'b': 1, 'c': 2}}",
			context: map[string]interface{}{},
			want:    map[string]interface{}{"a": map[string]interface{}{"b": 1, "c": 2}},
		},
		{
			name:    "direct access to literal - dict",
			expr:    "{'a': 1}['a']",
			context: map[string]interface{}{},
			want:    1,
		},
		{
			name:    "direct access to literal - list",
			expr:    "[10, 20, 30][1]",
			context: map[string]interface{}{},
			want:    20,
		},
		{
			name:    "compound subscript on literal nested dict",
			expr:    "{'level1': {'level2': {'level3': {'value': 42}}}}['level1']['level2']['level3']['value']",
			context: map[string]interface{}{},
			want:    42,
		},
		{
			name:    "compound subscript on literal mixed dict and list",
			expr:    "{'users': [{'name': 'Alice', 'age': 30}, {'name': 'Bob', 'age': 25}]}['users'][1]['name']",
			context: map[string]interface{}{},
			want:    "Bob",
		},
		{
			name:    "compound subscript on literal list with dict",
			expr:    "[{'a': 1}, {'a': 2}, {'a': 3}][1]['a']",
			context: map[string]interface{}{},
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evaluateCompoundExpression(tt.expr, tt.context)

			if (err != nil) != tt.wantError {
				t.Errorf("evaluateCompoundExpression() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError {
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("evaluateCompoundExpression() = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}

func TestLALRParser(t *testing.T) {
	tests := []struct {
		name     string
		expr     string
		context  map[string]interface{}
		expected interface{}
		wantErr  bool
	}{
		{
			name:     "Simple variable",
			expr:     "foo",
			context:  map[string]interface{}{"foo": "bar"},
			expected: "bar",
			wantErr:  false,
		},
		{
			name:     "Simple arithmetic",
			expr:     "1 + 2 * 3",
			context:  map[string]interface{}{},
			expected: 7,
			wantErr:  false,
		},
		{
			name:     "Operator precedence",
			expr:     "2 + 3 * 4 + 5",
			context:  map[string]interface{}{},
			expected: 19,
			wantErr:  false,
		},
		{
			name:     "Parenthesized expression",
			expr:     "(2 + 3) * 4",
			context:  map[string]interface{}{},
			expected: 20,
			wantErr:  false,
		},
		{
			name:     "Nested complex expression",
			expr:     "10 + 2 * (3 + 4 * (5 - 2))",
			context:  map[string]interface{}{},
			expected: 40,
			wantErr:  false,
		},
		{
			name:     "Variable in expression",
			expr:     "count * 2 + offset",
			context:  map[string]interface{}{"count": 5, "offset": 3},
			expected: 13,
			wantErr:  false,
		},
		{
			name:     "Dictionary literal",
			expr:     "{'name': 'Alice', 'age': 30}",
			context:  map[string]interface{}{},
			expected: map[string]interface{}{"name": "Alice", "age": 30},
			wantErr:  false,
		},
		{
			name:     "List literal",
			expr:     "[1, 2, 3, 4]",
			context:  map[string]interface{}{},
			expected: []interface{}{1, 2, 3, 4},
			wantErr:  false,
		},
		{
			name:     "Complex compound access",
			expr:     "{'users': [{'name': 'Alice', 'age': 30}, {'name': 'Bob', 'age': 25}]}['users'][0]['name']",
			context:  map[string]interface{}{},
			expected: "Alice",
			wantErr:  false,
		},
		{
			name:     "Nested dictionary",
			expr:     "config['server']['host']",
			context:  map[string]interface{}{"config": map[string]interface{}{"server": map[string]interface{}{"host": "localhost"}}},
			expected: "localhost",
			wantErr:  false,
		},
		{
			name:     "Logic operators short-circuit - AND",
			expr:     "false and undefined_var",
			context:  map[string]interface{}{"false": false},
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Logic operators short-circuit - OR",
			expr:     "true or undefined_var",
			context:  map[string]interface{}{"true": true},
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Unary operators",
			expr:     "not False",
			context:  map[string]interface{}{},
			expected: true,
			wantErr:  false,
		},
		{
			name:     "Negative number",
			expr:     "-42",
			context:  map[string]interface{}{},
			expected: -42,
			wantErr:  false,
		},
		{
			name:    "Syntax error",
			expr:    "1 + * 2",
			context: map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "Undefined variable",
			expr:    "undefined_var",
			context: map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test our new LALR parser
			got, err := ParseAndEvaluate(tt.expr, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("LALR Parser: error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(got, tt.expected) {
					t.Errorf("LALR Parser: got = %v, want %v", got, tt.expected)
				}
			}

			// Compare with EvaluateExpression result for compatibility
			got2, err := EvaluateExpression(tt.expr, tt.context)
			if (err != nil) != tt.wantErr {
				t.Logf("Note: Original EvaluateExpression got error = %v, but LALR parser had wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr && !reflect.DeepEqual(got, got2) {
				t.Logf("Note: LALR parser got = %v, but original EvaluateExpression got = %v", got, got2)
			}
		})
	}
}
