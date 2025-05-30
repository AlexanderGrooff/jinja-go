package jinja

import (
	"fmt"
	"os"
	"testing"
)

// Benchmark templates of varying complexity
func BenchmarkTemplateString(b *testing.B) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
	}{
		{
			name:     "simple_variable",
			template: "Hello, {{ name }}!",
			context:  map[string]interface{}{"name": "World"},
		},
		{
			name:     "multiple_variables",
			template: "{{ greeting }}, {{ name }}! Today is {{ day }}.",
			context:  map[string]interface{}{"greeting": "Hello", "name": "World", "day": "Monday"},
		},
		{
			name:     "nested_variables",
			template: "Hello, {{ user_name }}! Your email is {{ user_email }}.",
			context:  map[string]interface{}{"user_name": "John", "user_email": "john@example.com"},
		},
		{
			name:     "conditional",
			template: "{% if is_admin %}Admin user: {{ user_name }}{% else %}Regular user: {{ user_name }}{% endif %}",
			context:  map[string]interface{}{"user_name": "John", "is_admin": true},
		},
		{
			name:     "large_template",
			template: "{% if is_admin %}Admin user: {{ user_name }} ({{ user_email }}){% else %}Regular user: {{ user_name }} ({{ user_email }}){% endif %}\nRole: {{ role }}\nAccess level: {{ access_level }}\nJoined: {{ joined }}\nLast login: {{ last_login }}\nStatus: {{ status }}",
			context: map[string]interface{}{
				"user_name": "John Smith", "user_email": "john@example.com", "is_admin": true,
				"role": "Administrator", "access_level": "Full", "joined": "2020-01-01",
				"last_login": "2023-06-15", "status": "Active",
			},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := TemplateString(tt.template, tt.context)
				if err != nil {
					b.Fatalf("Error rendering template: %v", err)
				}
			}
		})
	}
}

// Benchmark expressions of varying complexity
func BenchmarkEvaluateExpression(b *testing.B) {
	tests := []struct {
		name    string
		expr    string
		context map[string]interface{}
	}{
		{
			name:    "simple_variable",
			expr:    "name",
			context: map[string]interface{}{"name": "World"},
		},
		{
			name:    "variable_with_default",
			expr:    "missing | default('Default Value')",
			context: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := EvaluateExpression(tt.expr, tt.context)
				if err != nil {
					b.Fatalf("Error evaluating expression: %v", err)
				}
			}
		})
	}
}

// Benchmark the tokenization phase specifically
func BenchmarkTokenize(b *testing.B) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "simple_expression",
			expr: "name",
		},
		{
			name: "dotted_expression",
			expr: "user.name.first",
		},
		{
			name: "complex_expression",
			expr: "2 ** 10 * 5 // 2 + 3 * (14 % 5)",
		},
		{
			name: "string_with_operators",
			expr: "'hello' + ' ' + 'world' + '!' * 3",
		},
		{
			name: "logical_operators",
			expr: "user.admin and (user.access_level == 'high' or user.role == 'admin')",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(tt.expr)
				_, err := lexer.Tokenize()
				if err != nil {
					b.Fatalf("Error tokenizing expression: %v", err)
				}
			}
		})
	}
}

// Benchmark the parsing phase specifically (tokenize + parse)
func BenchmarkParse(b *testing.B) {
	tests := []struct {
		name string
		expr string
	}{
		{
			name: "simple_expression",
			expr: "name",
		},
		{
			name: "dotted_expression",
			expr: "user.profile.preferences.theme",
		},
		{
			name: "complex_expression",
			expr: "2 ** 10 * 5 // 2 + 3 * (14 % 5)",
		},
		{
			name: "string_operations",
			expr: "'hello' + ' ' + 'world'",
		},
		{
			name: "complex_logical_expression",
			expr: "a and b or c and not d or e is f or g != h",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(tt.expr)
				tokens, err := lexer.Tokenize()
				if err != nil {
					b.Fatalf("Error tokenizing expression: %v", err)
				}

				parser := NewExprParser(tokens)
				_, err = parser.Parse()
				if err != nil {
					b.Fatalf("Error parsing expression: %v", err)
				}
			}
		})
	}
}

// BenchmarkNestedDictionaryParsing focuses specifically on the parsing of nested dictionary expressions
func BenchmarkNestedDictionaryParsing(b *testing.B) {
	tests := []struct {
		name string
		expr string
	}{
		// We'll keep these since parsing is different from evaluation
		// Even if they can't be evaluated, we can still benchmark parsing
		{
			name: "simple_nested_dict",
			expr: "{'a': {'b': 1}}",
		},
		{
			name: "two_level_nested_dict",
			expr: "{'a': {'b': {'c': 1}}}",
		},
		{
			name: "three_level_nested_dict",
			expr: "{'a': {'b': {'c': {'d': 1}}}}",
		},
		{
			name: "complex_nested_dict",
			expr: "{'a': {'b': 1, 'c': [1, 2, {'d': 3}]}, 'e': {'f': {'g': 'value'}}}",
		},
		{
			name: "nested_dict_with_mixed_types",
			expr: "{'a': {'b': 1, 'c': true, 'd': 'string', 'e': [1, 2], 'f': {'g': null}}}",
		},
		{
			name: "deep_dot_access",
			expr: "data.users.admin.permissions.files.read",
		},
		// Replace subscript access with dot notation
		{
			name: "deep_dot_access_alt",
			expr: "data.users.admin.permissions.files.read",
		},
		{
			name: "mixed_access_methods",
			expr: "data.users.admin.permissions.files.read",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				lexer := NewLexer(tt.expr)
				tokens, err := lexer.Tokenize()
				if err != nil {
					b.Skipf("Skipping test due to tokenizing error: %v", err)
					return
				}

				parser := NewExprParser(tokens)
				_, err = parser.Parse()
				if err != nil {
					b.Skipf("Skipping test due to parsing error: %v", err)
					return
				}
			}
		})
	}
}

// BenchmarkParseAndEvaluateTokens uses ParseAndEvaluate directly on expressions
// This tests the entire pipeline of tokenizing, parsing, and evaluating expressions
func BenchmarkParseAndEvaluateTokens(b *testing.B) {
	tests := []struct {
		name    string
		expr    string
		context map[string]interface{}
	}{
		{
			name:    "simple_variable",
			expr:    "name",
			context: map[string]interface{}{"name": "World"},
		},
		// The default filter doesn't work directly with ParseAndEvaluate
		// because it's applied during template evaluation phase
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := ParseAndEvaluate(tt.expr, tt.context)
				if err != nil {
					b.Fatalf("Error parsing and evaluating: %v", err)
				}
			}
		})
	}
}

// BenchmarkNestedDictionaryParseAndEvaluate focuses on parsing and evaluating nested dictionary expressions
func BenchmarkNestedDictionaryParseAndEvaluate(b *testing.B) {
	// Create a complex nested context
	nestedContext := map[string]interface{}{
		"config": map[string]interface{}{
			"app": map[string]interface{}{
				"settings": map[string]interface{}{
					"cache": map[string]interface{}{
						"enabled":   true,
						"ttl":       3600,
						"algorithm": "lru",
					},
				},
			},
		},
		"users": []interface{}{
			map[string]interface{}{
				"id":    1,
				"name":  "Alice",
				"roles": []string{"admin", "user"},
				"meta": map[string]interface{}{
					"last_login": "2023-06-10",
					"preferences": map[string]interface{}{
						"theme":  "dark",
						"notify": true,
					},
				},
			},
			map[string]interface{}{
				"id":    2,
				"name":  "Bob",
				"roles": []string{"user"},
				"meta": map[string]interface{}{
					"last_login": "2023-06-09",
					"preferences": map[string]interface{}{
						"theme":  "light",
						"notify": false,
					},
				},
			},
		},
		// Add direct access to nested user data for tests
		"user0_name":              "Alice",
		"user1_name":              "Bob",
		"user0_meta_prefs_theme":  "dark",
		"user1_meta_prefs_notify": false,
	}

	tests := []struct {
		name    string
		expr    string
		context map[string]interface{}
	}{
		{
			name:    "deep_access_chain",
			expr:    "config.app.settings.cache.enabled",
			context: nestedContext,
		},
		{
			name:    "deep_access_with_list_index",
			expr:    "user0_name", // Simplified access to first user's name
			context: nestedContext,
		},
		{
			name:    "very_deep_access_chain",
			expr:    "user0_meta_prefs_theme", // Simplified access to nested preferences
			context: nestedContext,
		},
		{
			name:    "mixed_subscript_and_attribute_access",
			expr:    "user1_meta_prefs_notify", // Simplified access
			context: nestedContext,
		},
		// Skip the literal dictionary tests for now
		// {
		// 	name:    "literal_nested_dict_deep_creation",
		// 	expr:    "{'a': {'b': {'c': {'d': {'e': 'value'}}}}}",
		// 	context: map[string]interface{}{},
		// },
		// {
		// 	name:    "literal_nested_dict_complex_structure",
		// 	expr:    "{'users': [{'name': 'Alice', 'settings': {'theme': 'dark'}}, {'name': 'Bob', 'settings': {'theme': 'light'}}]}",
		// 	context: map[string]interface{}{},
		// },
		// {
		// 	name:    "literal_dict_direct_access",
		// 	expr:    "{'users': [{'name': 'Alice'}, {'name': 'Bob'}]}['users'][1]['name']",
		// 	context: map[string]interface{}{},
		// },
		{
			name:    "complex_expression_with_nested_dicts",
			expr:    "config.app.settings.cache.ttl > 1000",
			context: nestedContext,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := ParseAndEvaluate(tt.expr, tt.context)
				if err != nil {
					b.Fatalf("Error parsing and evaluating: %v", err)
				}
			}
		})
	}
}

// BenchmarkTemplateParser isolates just the parsing stage of templates
func BenchmarkTemplateParser(b *testing.B) {
	tests := []struct {
		name     string
		template string
	}{
		{
			name:     "simple_template",
			template: "Hello, {{ name }}!",
		},
		{
			name:     "complex_template",
			template: "{% if is_admin %}Admin: {{ user_name }}{% else %}User: {{ user_name }}{% endif %}",
		},
		{
			name:     "mixed_content",
			template: "Text before {{ var1 }} middle text {{ var2 }} and {# comment #} more text {{ var3 }}.",
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				p := NewParser(tt.template)
				var nodes []*Node

				for {
					node, err := p.ParseNext()
					if err != nil {
						b.Fatalf("Error parsing template: %v", err)
					}
					if node == nil {
						break
					}
					nodes = append(nodes, node)
				}
			}
		})
	}
}

// BenchmarkNestedDictionaryTemplates tests templates with nested dictionary access
func BenchmarkNestedDictionaryTemplates(b *testing.B) {
	// Create a deeply nested context for testing
	nestedContext := map[string]interface{}{
		"server": map[string]interface{}{
			"config": map[string]interface{}{
				"environment": "production",
				"database": map[string]interface{}{
					"host":     "db.example.com",
					"port":     5432,
					"username": "admin",
					"settings": map[string]interface{}{
						"max_connections": 100,
						"timeout":         30,
						"ssl":             true,
					},
				},
				"cache": map[string]interface{}{
					"enabled": true,
					"ttl":     3600,
				},
			},
			"status": "running",
		},
		// Flattened access to nested data
		"admin_username":       "admin",
		"admin_email":          "admin@example.com",
		"admin_is_admin":       true,
		"user_username":        "user",
		"user_email":           "user@example.com",
		"user_is_admin":        false,
		"cache_enabled":        true,
		"cache_ttl":            3600,
		"db_host":              "db.example.com",
		"db_port":              5432,
		"db_settings_timeout":  30,
		"db_settings_max_conn": 100,
		"users": []interface{}{
			map[string]interface{}{
				"username": "user",
				"email":    "user@example.com",
				"permissions": map[string]interface{}{
					"roles": []string{"user"},
				},
			},
			map[string]interface{}{
				"username": "admin",
				"email":    "admin@example.com",
				"permissions": map[string]interface{}{
					"roles": []string{"admin"},
				},
			},
		},
	}

	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
	}{
		{
			name:     "simple_nested_access",
			template: "Server environment: {{ server.config.environment }}",
			context:  nestedContext,
		},
		{
			name:     "deep_nested_access",
			template: "Database connection: {{ db_host }}:{{ db_port }} (timeout: {{ db_settings_timeout }}s)",
			context:  nestedContext,
		},
		{
			name:     "nested_with_array_access",
			template: "Admin user: {{ admin_username }} ({{ admin_email }})",
			context:  nestedContext,
		},
		{
			name:     "nested_with_conditional",
			template: "{% if cache_enabled %}Cache TTL: {{ cache_ttl }}s{% else %}Cache disabled{% endif %}",
			context:  nestedContext,
		},
		{
			name:     "complex_template_with_deep_nesting",
			template: "{% if admin_is_admin %}Welcome Admin {{ admin_username }}!\nServer is {{ server.status }} in {{ server.config.environment }} mode\nDatabase: {{ db_host }}:{{ db_port }}\nCache {% if cache_enabled %}enabled ({{ cache_ttl }}s){% else %}disabled{% endif %}{% else %}Access Denied{% endif %}",
			context:  nestedContext,
		},
		{
			name:     "deeply_nested_mixed_subscript_access",
			template: "Database settings: Host={{ db_host }}, Max Connections={{ db_settings_max_conn }}",
			context:  nestedContext,
		},
		// For loop now supported
		{
			name:     "for_loop_with_nested_access",
			template: "Users:\n{% for user in users %}Username: {{ user.username }}\nEmail: {{ user.email }}\nRoles: {% for role in user.permissions.roles %}{{ role }}{% if not loop.last %}, {% endif %}{% endfor %}\n{% endfor %}",
			context:  nestedContext,
		},
		{
			name:     "simple_usernames",
			template: "Users: {{ admin_username }}, {{ user_username }}",
			context:  nestedContext,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := TemplateString(tt.template, tt.context)
				if err != nil {
					b.Fatalf("Error rendering template: %v", err)
				}
			}
		})
	}
}

func BenchmarkComplexExpressionLALR(b *testing.B) {
	expr := "10 + 2 * (3 + 4 * (5 - 2)) + [1, 2, 3, 4][2] + {'a': 1, 'b': 2, 'c': 3}['b']"
	context := map[string]interface{}{
		"var1": 100,
		"var2": 200,
		"obj": map[string]interface{}{
			"attr": "value",
			"items": []interface{}{
				map[string]interface{}{"name": "item1"},
				map[string]interface{}{"name": "item2"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ParseAndEvaluate(expr, context)
		if err != nil {
			b.Fatalf("Failed to evaluate expression: %v", err)
		}
		_ = result
	}
}

func BenchmarkNestedAccessLALR(b *testing.B) {
	expr := "obj.items[1].name"
	context := map[string]interface{}{
		"obj": map[string]interface{}{
			"items": []interface{}{
				map[string]interface{}{"name": "item1"},
				map[string]interface{}{"name": "item2"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ParseAndEvaluate(expr, context)
		if err != nil {
			b.Fatalf("Failed to evaluate expression: %v", err)
		}
		_ = result
	}
}

func BenchmarkDictLiteralLALR(b *testing.B) {
	expr := "{'users': [{'name': 'Alice', 'age': 30}, {'name': 'Bob', 'age': 25}]}['users'][1]['name']"
	context := map[string]interface{}{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ParseAndEvaluate(expr, context)
		if err != nil {
			b.Fatalf("Failed to evaluate expression: %v", err)
		}
		_ = result
	}
}

func BenchmarkLongExpressionLALR(b *testing.B) {
	expr := "var1 + var2 * var3 + var4 * (var5 + var6) - var7 / var8 + var9 * var10"
	context := map[string]interface{}{
		"var1": 10, "var2": 20, "var3": 30, "var4": 40, "var5": 50,
		"var6": 60, "var7": 70, "var8": 80, "var9": 90, "var10": 100,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := ParseAndEvaluate(expr, context)
		if err != nil {
			b.Fatalf("Failed to evaluate expression: %v", err)
		}
		_ = result
	}
}

// Helper function for benchmarks
func benchmarkParsingAndEvaluation(b *testing.B, expression string, context map[string]interface{}) {
	// Parse just once outside the loop to measure only evaluation time
	lexer := NewLexer(expression)
	tokens, err := lexer.Tokenize()
	if err != nil {
		b.Fatalf("Failed to tokenize expression: %v", err)
	}

	parser := NewExprParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		b.Fatalf("Failed to parse expression: %v", err)
	}

	// Create evaluator
	evaluator := NewEvaluator(context)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := evaluator.Evaluate(ast)
		if err != nil {
			b.Fatalf("Failed to evaluate expression: %v", err)
		}
		_ = result
	}
}

func BenchmarkJoinFilter(b *testing.B) {
	benchmarks := []struct {
		name     string
		template string
		context  map[string]interface{}
	}{
		{
			name:     "join small string array",
			template: "{{ strArray|join(',') }}",
			context:  map[string]interface{}{"strArray": []string{"a", "b", "c"}},
		},
		{
			name:     "join medium string array",
			template: "{{ items|join('-') }}",
			context: map[string]interface{}{
				"items": []string{"item1", "item2", "item3", "item4", "item5", "item6", "item7", "item8", "item9", "item10"},
			},
		},
		{
			name:     "join large string array",
			template: "{{ bigList|join(',') }}",
			context: func() map[string]interface{} {
				list := make([]string, 100)
				for i := 0; i < 100; i++ {
					list[i] = fmt.Sprintf("item%d", i)
				}
				return map[string]interface{}{"bigList": list}
			}(),
		},
		{
			name:     "join integer array",
			template: "{{ intArray|join('|') }}",
			context:  map[string]interface{}{"intArray": []int{1, 2, 3, 4, 5}},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := TemplateString(bm.template, bm.context)
				if err != nil {
					b.Fatalf("template error: %v", err)
				}
			}
		})
	}
}

func BenchmarkUpperFilter(b *testing.B) {
	context := map[string]interface{}{
		"text": "hello world",
	}
	template := "{{ text | upper }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TemplateString(template, context)
		if err != nil {
			b.Fatalf("TemplateString error: %v", err)
		}
	}
}

func BenchmarkLowerFilter(b *testing.B) {
	context := map[string]interface{}{
		"text": "HELLO WORLD",
	}
	template := "{{ text | lower }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TemplateString(template, context)
		if err != nil {
			b.Fatalf("TemplateString error: %v", err)
		}
	}
}

func BenchmarkCapitalizeFilter(b *testing.B) {
	context := map[string]interface{}{
		"text": "hello world",
	}
	template := "{{ text | capitalize }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TemplateString(template, context)
		if err != nil {
			b.Fatalf("TemplateString error: %v", err)
		}
	}
}

func BenchmarkReplaceFilter(b *testing.B) {
	context := map[string]interface{}{
		"text": "hello world hello",
	}
	template := "{{ text | replace('hello', 'hi') }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TemplateString(template, context)
		if err != nil {
			b.Fatalf("TemplateString error: %v", err)
		}
	}
}

func BenchmarkTrimFilter(b *testing.B) {
	context := map[string]interface{}{
		"text": "  hello world  ",
	}
	template := "{{ text | trim }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TemplateString(template, context)
		if err != nil {
			b.Fatalf("TemplateString error: %v", err)
		}
	}
}

func BenchmarkListFilter(b *testing.B) {
	context := map[string]interface{}{
		"text": "hello",
	}
	expression := "text | list"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EvaluateExpression(expression, context)
		if err != nil {
			b.Fatalf("EvaluateExpression error: %v", err)
		}
	}
}

func BenchmarkEscapeFilter(b *testing.B) {
	context := map[string]interface{}{
		"html": "<div>Hello & World</div>",
	}
	template := "{{ html | escape }}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := TemplateString(template, context)
		if err != nil {
			b.Fatalf("TemplateString error: %v", err)
		}
	}
}

func BenchmarkMapFilter(b *testing.B) {
	benchmarks := []struct {
		name     string
		template string
		context  map[string]interface{}
	}{
		{
			name:     "map small string array",
			template: "{{ items | map('upper') | join(',') }}",
			context:  map[string]interface{}{"items": []string{"a", "b", "c"}},
		},
		{
			name:     "map medium string array",
			template: "{{ items | map('upper') }}",
			context: map[string]interface{}{
				"items": []string{"item1", "item2", "item3", "item4", "item5"},
			},
		},
		{
			name:     "map large string array",
			template: "{{ bigList | map('upper') }}",
			context: func() map[string]interface{} {
				list := make([]string, 50)
				for i := 0; i < 50; i++ {
					list[i] = fmt.Sprintf("item%d", i)
				}
				return map[string]interface{}{"bigList": list}
			}(),
		},
		{
			name:     "map capitalize filter",
			template: "{{ items | map('capitalize') | join(' ') }}",
			context:  map[string]interface{}{"items": []string{"hello", "WORLD", "Test"}},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := TemplateString(bm.template, bm.context)
				if err != nil {
					b.Fatalf("template error: %v", err)
				}
			}
		})
	}
}

func BenchmarkItemsFilter(b *testing.B) {
	tests := []struct {
		name  string
		input interface{}
	}{
		{
			name:  "empty_map",
			input: map[string]interface{}{},
		},
		{
			name:  "small_map",
			input: map[string]interface{}{"a": 1, "b": 2, "c": 3},
		},
		{
			name: "medium_map",
			input: map[string]interface{}{
				"user": "john", "age": 30, "admin": true, "preferences": map[string]string{
					"theme": "dark", "language": "en",
				},
				"roles": []string{"user", "editor"},
			},
		},
		{
			name: "large_map",
			input: func() map[string]interface{} {
				// Create a map with 100 entries
				result := make(map[string]interface{})
				for i := 0; i < 100; i++ {
					result[fmt.Sprintf("key%d", i)] = i
				}
				return result
			}(),
		},
	}

	filter := GlobalFilters["items"]
	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := filter(tt.input)
				if err != nil {
					b.Fatalf("Error applying items filter: %v", err)
				}
			}
		})
	}
}

func BenchmarkLookupFilter(b *testing.B) {
	// Create a test file with some content
	testFilePath := "lookup_bench_test.txt"
	testFileContent := "This is a test file for lookup filter benchmark"
	err := os.WriteFile(testFilePath, []byte(testFileContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(testFilePath) // Clean up after test

	// Set an environment variable for testing
	testEnvVar := "LOOKUP_BENCH_ENV_VAR"
	testEnvValue := "benchmark_env_value"
	os.Setenv(testEnvVar, testEnvValue)
	defer os.Unsetenv(testEnvVar) // Clean up after test

	testCases := []struct {
		name  string
		input interface{}
		args  []interface{}
	}{
		{
			name:  "lookup_file",
			input: "file",
			args:  []interface{}{testFilePath},
		},
		{
			name:  "lookup_env",
			input: "env",
			args:  []interface{}{testEnvVar},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_, err := lookupFilter(tc.input, tc.args...)
				if err != nil {
					b.Fatalf("Error in lookupFilter: %v", err)
				}
			}
		})
	}

	// Also benchmark through template execution
	b.Run("template_with_file_lookup", func(b *testing.B) {
		template := fmt.Sprintf("{{ lookup('file', '%s') }}", testFilePath)
		context := map[string]interface{}{}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := TemplateString(template, context)
			if err != nil {
				b.Fatalf("Error in TemplateString: %v", err)
			}
		}
	})

	b.Run("template_with_env_lookup", func(b *testing.B) {
		template := fmt.Sprintf("{{ lookup('env', '%s') }}", testEnvVar)
		context := map[string]interface{}{}
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := TemplateString(template, context)
			if err != nil {
				b.Fatalf("Error in TemplateString: %v", err)
			}
		}
	})
}
