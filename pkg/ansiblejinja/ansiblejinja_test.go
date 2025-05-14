package ansiblejinja

import (
	"reflect"
	"testing"
)

func TestTemplateString(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool // We don't expect errors from TemplateString based on current implementation
	}{
		{
			name:     "empty template",
			template: "",
			context:  map[string]interface{}{"name": "World"},
			want:     "",
		},
		{
			name:     "no variables",
			template: "Hello World!",
			context:  map[string]interface{}{},
			want:     "Hello World!",
		},
		{
			name:     "simple variable substitution",
			template: "Hello {{ name }}!",
			context:  map[string]interface{}{"name": "Jinja"},
			want:     "Hello Jinja!",
		},
		{
			name:     "variable with leading/trailing spaces in tag",
			template: "Hello {{  name  }}!",
			context:  map[string]interface{}{"name": "Jinja"},
			want:     "Hello Jinja!",
		},
		{
			name:     "multiple variables",
			template: "{{ greeting }} {{ name }}! Age: {{ age }}",
			context:  map[string]interface{}{"greeting": "Hi", "name": "Alex", "age": 30},
			want:     "Hi Alex! Age: 30",
		},
		{
			name:     "variable not in context",
			template: "Hello {{ name }}! Your city is {{ city }}.",
			context:  map[string]interface{}{"name": "User"},
			want:     "Hello User! Your city is .", // city becomes empty string
		},
		{
			name:     "context has unused variables",
			template: "Hello {{ name }}",
			context:  map[string]interface{}{"name": "There", "unused": "data"},
			want:     "Hello There",
		},
		{
			name:     "integer variable",
			template: "Count: {{ count }}",
			context:  map[string]interface{}{"count": 123},
			want:     "Count: 123",
		},
		{
			name:     "boolean variable",
			template: "Enabled: {{isEnabled}}",
			context:  map[string]interface{}{"isEnabled": true},
			want:     "Enabled: true",
		},
		{
			name:     "empty context",
			template: "Value: {{ val }}",
			context:  map[string]interface{}{},
			want:     "Value: ", // val becomes empty
		},
		{
			name:     "template with only a variable",
			template: "{{data}}",
			context:  map[string]interface{}{"data": "test123"},
			want:     "test123",
		},
		{
			name:     "unclosed open tag at end",
			template: "Hello {{",
			context:  map[string]interface{}{"name": "Test"},
			want:     "Hello {{", // Treated as literal
		},
		{
			name:     "unclosed open tag with content",
			template: "Hello {{ name",
			context:  map[string]interface{}{"name": "Test"},
			want:     "Hello {{ name", // Treated as literal
		},
		{
			name:     "unclosed open tag with partial close",
			template: "Hello {{ name }",
			context:  map[string]interface{}{"name": "Test"},
			want:     "Hello {{ name }", // Treated as literal
		},
		{
			name:     "valid tag followed by unclosed tag",
			template: "Hi {{user}} {{name",
			context:  map[string]interface{}{"user": "Alex", "name": "Test"},
			want:     "Hi Alex {{name",
		},
		{
			name:     "double open brace literal",
			template: "This is not a tag: { { name } }",
			context:  map[string]interface{}{"name": "Test"},
			want:     "This is not a tag: { { name } }",
		},
		{
			name:     "ansible error case for nested variables",
			template: "var1 {{ var1 }} and var2 {{ {{ var2 }} }}",
			context:  map[string]interface{}{"var1": "val1", "var2": "val2"},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "consecutive tags",
			template: "{{first}}{{second}}",
			context:  map[string]interface{}{"first": "1st", "second": "2nd"},
			want:     "1st2nd",
		},
		{
			name:     "empty expression tag",
			template: "Value: {{}}",
			context:  map[string]interface{}{},
			// According to current parser, content is "", trim is "", lookup fails, result is "Value: "
			// This matches many Jinja engines' behavior for {{ }} or {{  }} when key is empty string.
			want: "Value: ",
		},
		{
			name:     "expression tag with only spaces",
			template: "Value: {{   }}",
			context:  map[string]interface{}{},
			want:     "Value: ",
		},
		{
			name:     "string expression with escaped double quotes inside expression",
			template: "{{ \"{{ name }}\" }}",
			context:  map[string]interface{}{}, // name not needed as outer is literal
			want:     "{{ name }}",             // The literal string is rendered
		},
		// Default Filter Tests for TemplateString
		{
			name:     "template with default filter for undefined variable",
			template: "Hello {{ undefined_var | default('World') }}!",
			context:  map[string]interface{}{},
			want:     "Hello World!",
		},
		{
			name:     "template with default filter for defined variable",
			template: "Hello {{ name | default('Fallback') }}!",
			context:  map[string]interface{}{"name": "Jinja"},
			want:     "Hello Jinja!",
		},
		{
			name:     "template with default filter for empty string variable",
			template: "Value: {{ empty_val | default('DefaultValue') }}",
			context:  map[string]interface{}{"empty_val": ""},
			want:     "Value: DefaultValue",
		},
		{
			name:     "template with default filter for zero value variable",
			template: "Count: {{ num | default(100) }}",
			context:  map[string]interface{}{"num": 0},
			want:     "Count: 0",
		},
		{
			name:     "template with default filter for false variable",
			template: "Enabled: {{ flag | default(true) }}",
			context:  map[string]interface{}{"flag": false},
			want:     "Enabled: true",
		},
		{
			name:     "template with default filter with variable as default",
			template: "{{ undef | default(other_var) }}",
			context:  map[string]interface{}{"other_var": "DefaultFromVar"},
			want:     "DefaultFromVar",
		},
		{
			name:     "template with default filter with literal string default containing spaces",
			template: "{{ undef | default('Hello World') }}",
			context:  map[string]interface{}{},
			want:     "Hello World",
		},
		{
			name:     "template with default filter for nil value in context",
			template: "Value: {{ nil_val | default('IsNil') }}",
			context:  map[string]interface{}{"nil_val": nil},
			want:     "Value: IsNil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TemplateString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("TemplateString() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvaluateExpression(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		context    map[string]interface{}
		want       interface{}
		wantErr    bool
	}{
		{
			name:       "simple evaluation",
			expression: "name",
			context:    map[string]interface{}{"name": "Jinja"},
			want:       "Jinja",
			wantErr:    false,
		},
		{
			name:       "expression with leading/trailing spaces",
			expression: "  name  ",
			context:    map[string]interface{}{"name": "Jinja"},
			want:       "Jinja",
			wantErr:    false,
		},
		{
			name:       "integer evaluation",
			expression: "age",
			context:    map[string]interface{}{"age": 42},
			want:       42,
			wantErr:    false,
		},
		{
			name:       "boolean evaluation",
			expression: "isActive",
			context:    map[string]interface{}{"isActive": true},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "variable not in context",
			expression: "city",
			context:    map[string]interface{}{"name": "User"},
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "empty expression string",
			expression: "",
			context:    map[string]interface{}{"": "empty_key_value"},
			want:       "empty_key_value", // Assuming an empty string can be a key
			wantErr:    false,
		},
		{
			name:       "empty expression string, not in context",
			expression: "",
			context:    map[string]interface{}{"name": "User"},
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "expression with only spaces, key exists",
			expression: "   ",
			context:    map[string]interface{}{"": "space_key_value"}, // Trimmed key is ""
			want:       "space_key_value",
			wantErr:    false,
		},
		{
			name:       "expression with only spaces, key does not exist",
			expression: "   ",
			context:    map[string]interface{}{"realKey": "value"}, // Trimmed key is ""
			want:       nil,
			wantErr:    true,
		},
		// Default Filter Tests for EvaluateExpression
		{
			name:       "evaluate default filter for undefined variable",
			expression: "undefined_var | default('World')",
			context:    map[string]interface{}{},
			want:       "World",
			wantErr:    false,
		},
		{
			name:       "evaluate default filter for defined variable",
			expression: "name | default('Fallback')",
			context:    map[string]interface{}{"name": "Jinja"},
			want:       "Jinja",
			wantErr:    false,
		},
		{
			name:       "evaluate default filter for empty string variable",
			expression: "empty_val | default('DefaultValue')",
			context:    map[string]interface{}{"empty_val": ""},
			want:       "DefaultValue",
			wantErr:    false,
		},
		{
			name:       "evaluate default filter for zero value variable",
			expression: "num | default(100)",
			context:    map[string]interface{}{"num": 0},
			want:       0,
			wantErr:    false,
		},
		{
			name:       "evaluate default filter for false variable",
			expression: "flag | default(true)",
			context:    map[string]interface{}{"flag": false},
			want:       true,
			wantErr:    false,
		},
		{
			name:       "evaluate default filter with variable as default",
			expression: "undef | default(other_var)",
			context:    map[string]interface{}{"other_var": "DefaultFromVar"},
			want:       "DefaultFromVar",
			wantErr:    false,
		},
		{
			name:       "evaluate default filter with literal string default containing spaces",
			expression: "undef | default('Hello World')",
			context:    map[string]interface{}{},
			want:       "Hello World",
			wantErr:    false,
		},
		{
			name:       "evaluate default filter for nil value in context",
			expression: "nil_val | default('IsNil')",
			context:    map[string]interface{}{"nil_val": nil},
			want:       "IsNil",
			wantErr:    false,
		},
		// EvaluateExpression specific error case for undefined without default
		{
			name:       "evaluate strictly undefined variable (error case for EvaluateExpression)",
			expression: "strictly_undefined_var",
			context:    map[string]interface{}{},
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EvaluateExpression(tt.expression, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("EvaluateExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// reflect.DeepEqual is important for comparing interface{} values, especially slices/maps if they were used.
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("EvaluateExpression() got = %v (%T), want %v (%T)", got, got, tt.want, tt.want)
			}
		})
	}
}
