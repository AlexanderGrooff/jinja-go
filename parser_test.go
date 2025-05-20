package ansiblejinja

import (
	"reflect"
	"testing"
)

func TestParser_ParseNext(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []*Node
		wantErr  bool // For ParseNext, errors are not expected from the current implementation unless explicitly set
	}{
		{
			name:     "empty string",
			template: "",
			want:     nil,
		},
		{
			name:     "only literal text",
			template: "Hello, world!",
			want:     []*Node{{Type: NodeText, Content: "Hello, world!"}},
		},
		{
			name:     "simple expression",
			template: "{{ name }}",
			want:     []*Node{{Type: NodeExpression, Content: " name "}},
		},
		{
			name:     "literal then expression",
			template: "Hello {{ name }}",
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeExpression, Content: " name "},
			},
		},
		{
			name:     "expression then literal",
			template: "{{ name }}!",
			want: []*Node{
				{Type: NodeExpression, Content: " name "},
				{Type: NodeText, Content: "!"},
			},
		},
		{
			name:     "literal, expression, literal",
			template: "Hello {{ name }}!",
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeExpression, Content: " name "},
				{Type: NodeText, Content: "!"},
			},
		},
		{
			name:     "multiple expressions",
			template: "{{ greeting }} {{ name }}!",
			want: []*Node{
				{Type: NodeExpression, Content: " greeting "},
				{Type: NodeText, Content: " "},
				{Type: NodeExpression, Content: " name "},
				{Type: NodeText, Content: "!"},
			},
		},
		{
			name:     "consecutive expressions",
			template: "{{first}}{{second}}",
			want: []*Node{
				{Type: NodeExpression, Content: "first"},
				{Type: NodeExpression, Content: "second"},
			},
		},
		{
			name:     "unclosed tag at end of string",
			template: "Hello {{",
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeText, Content: "{{"},
			},
		},
		{
			name:     "unclosed tag with content at end of string",
			template: "Hello {{ name",
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeText, Content: "{{ name"},
			},
		},
		{
			name:     "unclosed tag in middle of string (same as content at end for current parser)",
			template: "Hello {{ name and goodbye",
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeText, Content: "{{ name and goodbye"},
			},
		},
		{
			name:     "unclosed tag followed by another literal (same as content at end)",
			template: "text {{ unclosed text_after",
			want: []*Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{{ unclosed text_after"},
			},
		},
		{
			name:     "double open brace literal, not a tag",
			template: "This has { { not a tag } }",
			want:     []*Node{{Type: NodeText, Content: "This has { { not a tag } }"}},
		},
		{
			name:     "expression with no spaces",
			template: "{{name}}",
			want:     []*Node{{Type: NodeExpression, Content: "name"}},
		},
		{
			name:     "expression with leading space",
			template: "{{ name}}",
			want:     []*Node{{Type: NodeExpression, Content: " name"}},
		},
		{
			name:     "expression with trailing space",
			template: "{{name }}",
			want:     []*Node{{Type: NodeExpression, Content: "name "}},
		},
		{
			name:     "empty expression",
			template: "{{}}",
			want:     []*Node{{Type: NodeExpression, Content: ""}},
		},
		{
			name:     "expression with only spaces",
			template: "{{   }}",
			want:     []*Node{{Type: NodeExpression, Content: "   "}},
		},
		{
			name:     "string with only {{ }} then text",
			template: "{{}} world",
			want: []*Node{
				{Type: NodeExpression, Content: ""},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:     "text then {{}} then text",
			template: "hello{{}}world",
			want: []*Node{
				{Type: NodeText, Content: "hello"},
				{Type: NodeExpression, Content: ""},
				{Type: NodeText, Content: "world"},
			},
		},
		{
			name:     "incomplete open tag at very end {{ a",
			template: "test {{ a",
			want: []*Node{
				{Type: NodeText, Content: "test "},
				{Type: NodeText, Content: "{{ a"},
			},
		},
		{
			name:     "incomplete open tag at very end {",
			template: "test {",
			want:     []*Node{{Type: NodeText, Content: "test {"}},
		},
		{
			name:     "template starting with unclosed tag",
			template: "{{ unfinished then text",
			want:     []*Node{{Type: NodeText, Content: "{{ unfinished then text"}},
		},
		{
			name:     "template with {{ and }} but content missing",
			template: "{{}} and {{ value }}",
			want: []*Node{
				{Type: NodeExpression, Content: ""},
				{Type: NodeText, Content: " and "},
				{Type: NodeExpression, Content: " value "},
			},
		},
		{
			name:     "only an unclosed tag {{tag",
			template: "{{tag",
			want:     []*Node{{Type: NodeText, Content: "{{tag"}},
		},
		{
			name:     "only {{ at end",
			template: "text{{",
			want: []*Node{
				{Type: NodeText, Content: "text"},
				{Type: NodeText, Content: "{{"},
			},
		},
		{
			name:     "string expression inside expression",
			template: "{{ 'some string' }}",
			want:     []*Node{{Type: NodeExpression, Content: " 'some string' "}},
		},
		{
			name:     "expression opening braces inside expression",
			template: "{{ '{{' }}",
			want:     []*Node{{Type: NodeExpression, Content: " '{{' "}},
		},
		{
			name:     "expression closing braces inside expression",
			template: "{{ '}}' }}",
			want:     []*Node{{Type: NodeExpression, Content: " '}}' "}},
		},
		{
			name:     "string expression with escaped braces inside expression",
			template: "{{ '{{}}' }}",
			want:     []*Node{{Type: NodeExpression, Content: " '{{}}' "}},
		},
		{
			name:     "string expression with escaped double quotes inside expression",
			template: "{{ '\"' }}",
			want:     []*Node{{Type: NodeExpression, Content: " '\"' "}},
		},
		{
			name:     "escaped nested expression",
			template: "{{ \"{{ name }}\" }}",
			want:     []*Node{{Type: NodeExpression, Content: " \"{{ name }}\" "}},
		},
		{
			name:     "nested {{ in string in expr then }}",
			template: "{{ \"a{{b\" }}",
			want:     []*Node{{Type: NodeExpression, Content: " \"a{{b\" "}},
		},
		{
			name:     "nested {{ in string in expr then }} then text",
			template: "{{ \"a{{b\" }} world",
			want: []*Node{
				{Type: NodeExpression, Content: " \"a{{b\" "},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:     "complex nested {{ and }} within strings",
			template: "Hello {{ \"a {{ b }} c\" | filter(\"x {{ y }} z\") }} Bye",
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeExpression, Content: " \"a {{ b }} c\" | filter(\"x {{ y }} z\") "},
				{Type: NodeText, Content: " Bye"},
			},
		},
		{
			name:     "simple comment",
			template: "{# this is a comment #}",
			want:     []*Node{{Type: NodeComment, Content: " this is a comment "}},
		},
		{
			name:     "comment with text before and after",
			template: "hello {# comment #} world",
			want: []*Node{
				{Type: NodeText, Content: "hello "},
				{Type: NodeComment, Content: " comment "},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:     "unclosed comment",
			template: "hello {# comment world",
			want:     []*Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{# comment world"}},
		},
		{
			name:     "simple control tag - if",
			template: "{% if condition %}",
			want:     []*Node{{Type: NodeControlTag, Content: "if condition", Control: &ControlTagInfo{Type: ControlIf, Expression: "condition"}}},
		},
		{
			name:     "simple control tag - endif",
			template: "{% endif %}",
			want:     []*Node{{Type: NodeControlTag, Content: "endif", Control: &ControlTagInfo{Type: ControlEndIf}}},
		},
		{
			name:     "control tag - if with extra spaces",
			template: "{%   if    condition   %}",
			want:     []*Node{{Type: NodeControlTag, Content: "if    condition", Control: &ControlTagInfo{Type: ControlIf, Expression: "condition"}}},
		},
		{
			name:     "control tag - if with complex condition",
			template: "{% if user.name == 'test' and user.age > 30 %}",
			want:     []*Node{{Type: NodeControlTag, Content: "if user.name == 'test' and user.age > 30", Control: &ControlTagInfo{Type: ControlIf, Expression: "user.name == 'test' and user.age > 30"}}},
		},
		{
			name:     "control tag - if missing condition",
			template: "{% if %}", // Parser creates an Unknown type node with error in Expression
			want:     []*Node{{Type: NodeControlTag, Content: "if", Control: &ControlTagInfo{Type: ControlUnknown, Expression: "Error parsing tag 'if': if tag requires a condition, e.g., {% if user.isAdmin %}"}}},
		},
		{
			name:     "control tag - endif with extra content",
			template: "{% endif this %}", // Parser creates an Unknown type node with error in Expression
			want:     []*Node{{Type: NodeControlTag, Content: "endif this", Control: &ControlTagInfo{Type: ControlUnknown, Expression: "Error parsing tag 'endif this': endif tag does not take any arguments, e.g., {% endif %}"}}},
		},
		{
			name:     "simple control tag - else",
			template: "{% else %}",
			want:     []*Node{{Type: NodeControlTag, Content: "else", Control: &ControlTagInfo{Type: ControlElse}}},
		},
		{
			name:     "control tag - else with arguments (invalid)",
			template: "{% else somearg %}",
			want:     []*Node{{Type: NodeControlTag, Content: "else somearg", Control: &ControlTagInfo{Type: ControlUnknown, Expression: "Error parsing tag 'else somearg': else tag does not take any arguments, e.g., {% else %}"}}},
		},
		{
			name:     "simple control tag - elif",
			template: "{% elif condition2 %}",
			want:     []*Node{{Type: NodeControlTag, Content: "elif condition2", Control: &ControlTagInfo{Type: ControlElseIf, Expression: "condition2"}}},
		},
		{
			name:     "control tag - elif with complex condition",
			template: "{% elif item.value == 100 or is_admin %}",
			want:     []*Node{{Type: NodeControlTag, Content: "elif item.value == 100 or is_admin", Control: &ControlTagInfo{Type: ControlElseIf, Expression: "item.value == 100 or is_admin"}}},
		},
		{
			name:     "control tag - elif missing condition (invalid)",
			template: "{% elif %}",
			want:     []*Node{{Type: NodeControlTag, Content: "elif", Control: &ControlTagInfo{Type: ControlUnknown, Expression: "Error parsing tag 'elif': elif tag requires a condition, e.g., {% elif user.isGuest %}"}}},
		},
		{
			name:     "unclosed control tag - if",
			template: "text {% if condition",
			want: []*Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{% if condition"},
			},
		},
		{
			name:     "control tag containing string with percent brace",
			template: "{% if name == \"test %}\" %}",
			want:     []*Node{{Type: NodeControlTag, Content: "if name == \"test %}\"", Control: &ControlTagInfo{Type: ControlIf, Expression: "name == \"test %}\""}}},
		},
		{
			name:     "mixed tags including control",
			template: "Value: {{ val }} {# comment #} {% if debug %}Debug Mode{% endif %}",
			want: []*Node{
				{Type: NodeText, Content: "Value: "},
				{Type: NodeExpression, Content: " val "},
				{Type: NodeText, Content: " "},
				{Type: NodeComment, Content: " comment "},
				{Type: NodeText, Content: " "},
				{Type: NodeControlTag, Content: "if debug", Control: &ControlTagInfo{Type: ControlIf, Expression: "debug"}},
				{Type: NodeText, Content: "Debug Mode"},
				{Type: NodeControlTag, Content: "endif", Control: &ControlTagInfo{Type: ControlEndIf}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.template)
			var got []*Node
			var currentErr error

			for {
				node, err := p.ParseNext()
				if err != nil {
					currentErr = err
					break
				}
				if node == nil { // EOF
					break
				}
				nodeCopy := *node        // Make a shallow copy for `got`
				if node.Control != nil { // Deep copy ControlTagInfo if present
					controlCopy := *node.Control
					nodeCopy.Control = &controlCopy
				}
				got = append(got, &nodeCopy)
			}

			if (currentErr != nil) != tt.wantErr {
				t.Errorf("p.ParseNext() error = %v, wantErr %v for template %q\n", currentErr, tt.wantErr, tt.template)
				return
			}
			if currentErr != nil && tt.wantErr {
				return
			}

			if len(tt.want) == 0 && len(got) == 0 {
				// Pass
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("p.ParseNext() for template %q produced incorrect nodes.\n", tt.template)
				t.Logf("Got (%d items):\n", len(got))
				for i, n := range got {
					t.Logf("  [%d] Type: %v, Content: %q, Control: %+v\n", i, n.Type, n.Content, n.Control)
				}
				t.Logf("Want (%d items):\n", len(tt.want))
				for i, n := range tt.want {
					t.Logf("  [%d] Type: %v, Content: %q, Control: %+v\n", i, n.Type, n.Content, n.Control)
				}
				t.FailNow()
			}
		})
	}
}

// TestParseNext_WithComplexInputs can be removed or merged if TestParser_ParseNext covers sufficiently.
// For now, keeping it separate if it targets different complexities or verbosity.
// If TestParser_ParseNext is comprehensive, this can be deprecated.
/*
func TestParseNext_WithComplexInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []*Node // Changed to slice of pointers
	}{
		{
			name:  "text, expression, text",
			input: "hello {{ name }} world",
			want: []*Node{
				{Type: NodeText, Content: "hello "},
				{Type: NodeExpression, Content: " name "},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:  "text with {{ and }} but not an expression due to spacing",
			input: "text { {var} } world",
			want:  []*Node{{Type: NodeText, Content: "text { {var} } world"}},
		},
		{
			name:  "Text only containing {{ but not closed",
			input: "{{",
			want:  []*Node{{Type: NodeText, Content: "{{"}},
		},
		{
			name:  "Text only containing {# but not closed",
			input: "{#",
			want:  []*Node{{Type: NodeText, Content: "{#"}},
		},
		{
			name:  "Text only containing {% but not closed",
			input: "{%",
			want:  []*Node{{Type: NodeText, Content: "{%"}},
		},
		{
			name:  "text, comment, expression, text, comment, expression",
			input: "A {# B #} C {{ D }} E {# F #} G {{ H }}",
			want: []*Node{
				{Type: NodeText, Content: "A "},
				{Type: NodeComment, Content: " B "},
				{Type: NodeText, Content: " C "},
				{Type: NodeExpression, Content: " D "},
				{Type: NodeText, Content: " E "},
				{Type: NodeComment, Content: " F "},
				{Type: NodeText, Content: " G "},
				{Type: NodeExpression, Content: " H "},
			},
		},
		{
			name:  "unclosed comment followed by valid expression",
			input: "text {# unclosed {{ expr }}",
			want: []*Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{# unclosed "},
				{Type: NodeExpression, Content: " expr "},
			},
		},
		{
			name:  "unclosed expr followed by valid comment",
			input: "text {{ unclosed {# comment #}",
			want: []*Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{{ unclosed "},
				{Type: NodeComment, Content: " comment "},
			},
		},
		{
			name:  "unclosed control tag followed by valid expression",
			input: "text {% unclosed {{ expr }}",
			want: []*Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{% unclosed "},
				{Type: NodeExpression, Content: " expr "},
			},
		},
		{
			name:  "unclosed expr followed by valid control tag",
			input: "text {{ unclosed {% if true %}",
			want: []*Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{{ unclosed "},
				{Type: NodeControlTag, Content: "if true", Control: &ControlTagInfo{Type:ControlIf, Expression: "true"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			var got []*Node // Changed to slice of pointers
			var currentErr error

			for {
				node, err := p.ParseNext()
				if err != nil {
					currentErr = err
					break
				}
				if node == nil { // EOF
					break
				}
				// node is already a pointer, just append it
				got = append(got, node)
			}

			if (currentErr != nil) != false {
				t.Errorf("p.ParseNext() error = %v, wantErr false for input %q\n", currentErr, tt.input)
				return
			}

			if len(tt.want) == 0 && len(got) == 0 {
				// Pass
			} else if !compareNodeSlices(got, tt.want) {
				t.Errorf("p.ParseNext() for input %q did not produce the expected nodes.\n", tt.input)
				t.Logf("Got (%d items):\n", len(got))
				for i, n := range got {
					t.Logf("  [%d] Type: %v, Content: %q, Control: %+v\n", i, n.Type, n.Content, n.Control)
				}
				t.Logf("Want (%d items):\n", len(tt.want))
				for i, n := range tt.want {
					t.Logf("  [%d] Type: %v, Content: %q, Control: %+v\n", i, n.Type, n.Content, n.Control)
				}
				t.FailNow()
			}
		})
	}
})
*/

// compareNodeSlices compares two slices of Node pointers for deep equality,
// paying special attention to the Control field.
func compareNodeSlices(got []*Node, want []*Node) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] == nil && want[i] == nil {
			continue
		}
		if got[i] == nil || want[i] == nil {
			return false // One is nil, the other is not
		}
		if got[i].Type != want[i].Type || got[i].Content != want[i].Content {
			return false
		}
		// Compare Control field
		if !reflect.DeepEqual(got[i].Control, want[i].Control) {
			return false
		}
	}
	return true
}
