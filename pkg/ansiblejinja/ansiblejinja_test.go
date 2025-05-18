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
		// Comment Tests for TemplateString
		{
			name:     "template with only a comment",
			template: "{# this is a comment #}",
			context:  map[string]interface{}{},
			want:     "",
		},
		{
			name:     "template with comment and text",
			template: "Hello {# comment #}World",
			context:  map[string]interface{}{},
			want:     "Hello World",
		},
		{
			name:     "template with comment at the beginning",
			template: "{# comment #}Hello World",
			context:  map[string]interface{}{},
			want:     "Hello World",
		},
		{
			name:     "template with comment at the end",
			template: "Hello World{# comment #}",
			context:  map[string]interface{}{},
			want:     "Hello World",
		},
		{
			name:     "template with multiple comments",
			template: "Text1 {# comment1 #}Text2{# comment2 #}Text3",
			context:  map[string]interface{}{},
			want:     "Text1 Text2Text3",
		},
		{
			name:     "template with comment and variable",
			template: "Value: {{ val }} {# this is a comment about val #}",
			context:  map[string]interface{}{"val": "test"},
			want:     "Value: test ",
		},
		{
			name:     "template with comment containing {{}}",
			template: "Data {# {{ fake_expr }} #}{{ real_val }} A{#B#}C",
			context:  map[string]interface{}{"real_val": "XYZ"},
			want:     "Data XYZ AC",
		},
		{
			name:     "unclosed comment",
			template: "Hello {# unclosed comment",
			context:  map[string]interface{}{},
			want:     "Hello {# unclosed comment", // Unclosed comment is treated as text by parser
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

func TestTemplateString_IfStatements(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  map[string]interface{}
		want     string
		wantErr  bool
	}{
		// Basic If/Endif
		{
			name:     "if true condition",
			template: "{% if true %}Hello{% endif %}",
			context:  map[string]interface{}{},
			want:     "Hello",
			wantErr:  false,
		},
		{
			name:     "if false condition",
			template: "{% if false %}Hello{% endif %}Bye",
			context:  map[string]interface{}{},
			want:     "Bye",
			wantErr:  false,
		},
		{
			name:     "if variable true",
			template: "{% if show %}Welcome{% endif %} User",
			context:  map[string]interface{}{"show": true},
			want:     "Welcome User",
			wantErr:  false,
		},
		{
			name:     "if variable false",
			template: "{% if show %}Welcome{% endif %} User",
			context:  map[string]interface{}{"show": false},
			want:     " User",
			wantErr:  false,
		},
		// Truthiness Tests
		{
			name:     "if empty string (falsey)",
			template: "{% if val %}Text{% endif %} End",
			context:  map[string]interface{}{"val": ""},
			want:     " End",
			wantErr:  false,
		},
		{
			name:     "if non-empty string (truthy)",
			template: "{% if val %}Text{% endif %} End",
			context:  map[string]interface{}{"val": "hello"},
			want:     "Text End",
			wantErr:  false,
		},
		{
			name:     "if zero int (falsey)",
			template: "{% if num %}Number{% endif %} Zero",
			context:  map[string]interface{}{"num": 0},
			want:     " Zero",
			wantErr:  false,
		},
		{
			name:     "if non-zero int (truthy)",
			template: "{% if num %}Number{% endif %} NonZero",
			context:  map[string]interface{}{"num": 1},
			want:     "Number NonZero",
			wantErr:  false,
		},
		{
			name:     "if nil value (falsey)",
			template: "{% if data %}Data{% endif %} End",
			context:  map[string]interface{}{"data": nil},
			want:     " End",
			wantErr:  false,
		},
		{
			name:     "if empty list (falsey)",
			template: "{% if items %}List{% endif %} Done",
			context:  map[string]interface{}{"items": []string{}},
			want:     " Done",
			wantErr:  false,
		},
		{
			name:     "if non-empty list (truthy)",
			template: "{% if items %}List{% endif %} Done",
			context:  map[string]interface{}{"items": []string{"a"}},
			want:     "List Done",
			wantErr:  false,
		},
		{
			name:     "if empty map (falsey)",
			template: "{% if dict %}Map{% endif %} EndMap",
			context:  map[string]interface{}{"dict": map[string]string{}},
			want:     " EndMap",
			wantErr:  false,
		},
		{
			name:     "if non-empty map (truthy)",
			template: "{% if dict %}Map{% endif %} EndMap",
			context:  map[string]interface{}{"dict": map[string]string{"key": "val"}},
			want:     "Map EndMap",
			wantErr:  false,
		},
		// Nested If Statements
		{
			name:     "nested if true true",
			template: "{% if outer %}Outer{% if inner %} Inner{% endif %} EndOuter{% endif %}",
			context:  map[string]interface{}{"outer": true, "inner": true},
			want:     "Outer Inner EndOuter",
			wantErr:  false,
		},
		{
			name:     "nested if true false",
			template: "{% if outer %}Outer{% if inner %} Inner{% endif %} EndOuter{% endif %}",
			context:  map[string]interface{}{"outer": true, "inner": false},
			want:     "Outer EndOuter",
			wantErr:  false,
		},
		{
			name:     "nested if false true (outer hides inner)",
			template: "{% if outer %}Outer{% if inner %} Inner{% endif %} EndOuter{% endif %}Rest",
			context:  map[string]interface{}{"outer": false, "inner": true},
			want:     "Rest",
			wantErr:  false,
		},
		// Error Cases for If Statements
		{
			name:     "unclosed if statement",
			template: "Hello {% if true %}Something",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "unexpected endif",
			template: "Hello {% endif %}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "if condition evaluates to undefined variable",
			template: "{% if undefined_var %}Text{% endif %}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "if tag missing condition",
			template: "{% if %}Hello{% endif %}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "if with expression inside then text (using pre-evaluated bool)",
			template: "{% if is_positive %}{{ val }} is positive.{% endif %} Val is {{ val }}",
			context:  map[string]interface{}{"val": 10, "is_positive": true},
			want:     "10 is positive. Val is 10",
			wantErr:  false,
		},
		{
			name:     "if with expression inside (false condition, using pre-evaluated bool)",
			template: "{% if is_negative %}{{ val }} is negative.{% endif %} Val is {{ val }}",
			context:  map[string]interface{}{"val": 10, "is_negative": false},
			want:     " Val is 10",
			wantErr:  false,
		},
		// If/Elif/Else/Endif tests
		{
			name:     "if true - else/elif not processed",
			template: "{% if cond1 %}A{% elif cond2 %}B{% else %}C{% endif %}D",
			context:  map[string]interface{}{"cond1": true, "cond2": true},
			want:     "AD",
			wantErr:  false,
		},
		{
			name:     "if false, elif true - else not processed",
			template: "{% if cond1 %}A{% elif cond2 %}B{% else %}C{% endif %}D",
			context:  map[string]interface{}{"cond1": false, "cond2": true},
			want:     "BD",
			wantErr:  false,
		},
		{
			name:     "if false, elif false, else processed",
			template: "{% if cond1 %}A{% elif cond2 %}B{% else %}C{% endif %}D",
			context:  map[string]interface{}{"cond1": false, "cond2": false},
			want:     "CD",
			wantErr:  false,
		},
		{
			name:     "if false, no elif, else processed",
			template: "{% if cond1 %}A{% else %}C{% endif %}D",
			context:  map[string]interface{}{"cond1": false},
			want:     "CD",
			wantErr:  false,
		},
		{
			name:     "multiple elif - first true",
			template: "{% if c1 %}1{% elif c2 %}2{% elif c3 %}3{% else %}4{% endif %}",
			context:  map[string]interface{}{"c1": false, "c2": true, "c3": true},
			want:     "2",
			wantErr:  false,
		},
		{
			name:     "multiple elif - second true",
			template: "{% if c1 %}1{% elif c2 %}2{% elif c3 %}3{% else %}4{% endif %}",
			context:  map[string]interface{}{"c1": false, "c2": false, "c3": true},
			want:     "3",
			wantErr:  false,
		},
		{
			name:     "multiple elif - none true, else processed",
			template: "{% if c1 %}1{% elif c2 %}2{% elif c3 %}3{% else %}4{% endif %}",
			context:  map[string]interface{}{"c1": false, "c2": false, "c3": false},
			want:     "4",
			wantErr:  false,
		},
		{
			name:     "if false, only elif, elif false, nothing rendered from block",
			template: "X{% if c1 %}A{% elif c2 %}B{% endif %}Y",
			context:  map[string]interface{}{"c1": false, "c2": false},
			want:     "XY",
			wantErr:  false,
		},
		{
			name:     "nested if-else constructs",
			template: "{% if o1 %}O1{% if i1 %}I1{% else %}I2{% endif %}{% else %}O2{% if i2 %}I3{% else %}I4{% endif %}{% endif %}",
			context:  map[string]interface{}{"o1": true, "i1": false, "i2": true},
			want:     "O1I2",
			wantErr:  false,
		},
		{
			name:     "nested if-else constructs - outer else branch",
			template: "{% if o1 %}O1{% if i1 %}I1{% else %}I2{% endif %}{% else %}O2{% if i2 %}I3{% else %}I4{% endif %}{% endif %}",
			context:  map[string]interface{}{"o1": false, "i1": true, "i2": false},
			want:     "O2I4",
			wantErr:  false,
		},
		// Error cases for else/elif
		{
			name:     "else outside if block",
			template: "Hello {% else %} World",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "elif outside if block",
			template: "Hello {% elif cond %} World",
			context:  map[string]interface{}{"cond": true},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "misplaced endif before else",
			template: "{% if true %}A{% endif %}{% else %}B{% endif %}",
			context:  map[string]interface{}{},
			want:     "", // First endif closes the if, second else is orphaned
			wantErr:  true,
		},
		{
			name:     "else with arguments (parser should mark as unknown, then rendering fails)",
			template: "{% if true %}A{% else badarg %}B{% endif %}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
		{
			name:     "elif without condition (parser should mark as unknown, then rendering fails)",
			template: "{% if false %}A{% elif %}B{% endif %}",
			context:  map[string]interface{}{},
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TemplateString(tt.template, tt.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateString() error = %v, wantErr %v for template %q", err, tt.wantErr, tt.template)
				return
			}
			if got != tt.want {
				t.Errorf("TemplateString() got = %q, want %q for template %q", got, tt.want, tt.template)
			}
		})
	}
}
