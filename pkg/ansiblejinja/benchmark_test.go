package ansiblejinja

import (
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

// Benchmark the Tokenize function to isolate lexing performance
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
				parser := NewExpressionParser(tt.expr)
				err := parser.tokenize()
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
				parser := NewExpressionParser(tt.expr)
				err := parser.tokenize()
				if err != nil {
					b.Fatalf("Error tokenizing expression: %v", err)
				}

				_, err = parser.parse()
				if err != nil {
					b.Fatalf("Error parsing expression: %v", err)
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
