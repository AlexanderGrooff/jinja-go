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
		wantErr  bool // For ParseNext, errors are not expected from the current implementation
	}{
		{
			name:     "empty string",
			template: "",
			want:     nil, // Or []*Node{} - nil is fine for an empty sequence.
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
			want: []*Node{ // New behavior: "Hello ", then "{{" as text
				{Type: NodeText, Content: "Hello "},
				{Type: NodeText, Content: "{{"},
			},
		},
		{
			name:     "unclosed tag with content at end of string",
			template: "Hello {{ name",
			want: []*Node{ // New behavior
				{Type: NodeText, Content: "Hello "},
				{Type: NodeText, Content: "{{ name"},
			},
		},
		{
			name:     "unclosed tag in middle of string (same as content at end for current parser)",
			template: "Hello {{ name and goodbye",
			want: []*Node{ // New behavior
				{Type: NodeText, Content: "Hello "},
				{Type: NodeText, Content: "{{ name and goodbye"},
			},
		},
		{
			name:     "unclosed tag followed by another literal (same as content at end)",
			template: "text {{ unclosed text_after",
			want: []*Node{ // New behavior
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
			want: []*Node{ // New behavior
				{Type: NodeText, Content: "test "},
				{Type: NodeText, Content: "{{ a"},
			},
		},
		{
			name:     "incomplete open tag at very end {",
			template: "test {", // This was treated as a single literal by old parser if no {{
			// New parser: this is just text, no '{{' encountered
			want: []*Node{{Type: NodeText, Content: "test {"}},
		},
		{
			name:     "template starting with unclosed tag",
			template: "{{ unfinished then text",
			// ParseNext will try to parse "{{ unfinished then text" as expression, fail,
			// then treat the whole thing as a single text node because the failure reset p.pos
			// and the subsequent text scan from p.pos will find no *further* "{{"
			want: []*Node{{Type: NodeText, Content: "{{ unfinished then text"}},
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
			// Similar to "template starting with unclosed tag"
			want: []*Node{{Type: NodeText, Content: "{{tag"}},
		},
		{
			name:     "only {{ at end",
			template: "text{{",
			want: []*Node{ // New behavior
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
			name:     "escaped nested expression", // Assuming escaped means within a string literal
			template: "{{ \"{{ name }}\" }}",      // The content is " \"{{ name }}\" "
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
			// parseExpressionTag should handle this correctly due to string literal skipping.
			// Content of NodeExpression: " \"a {{ b }} c\" | filter(\"x {{ y }} z\") "
			want: []*Node{
				{Type: NodeText, Content: "Hello "},
				{Type: NodeExpression, Content: " \"a {{ b }} c\" | filter(\"x {{ y }} z\") "},
				{Type: NodeText, Content: " Bye"},
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
				// Create a copy of the node to avoid issues if the parser reuses node memory (though it doesn't currently)
				// And to ensure we are comparing value semantics for 'want'
				nodeCopy := *node
				got = append(got, &nodeCopy)
			}

			if (currentErr != nil) != tt.wantErr {
				t.Errorf("p.ParseNext() error = %v, wantErr %v\n", currentErr, tt.wantErr)
				return
			}
			if currentErr != nil && tt.wantErr {
				return // Expected error occurred
			}

			// Handle empty 'want' specifically, as DeepEqual(got, nil) when got is empty slice is false.
			if len(tt.want) == 0 && len(got) == 0 {
				// This is a pass, both are effectively empty.
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("p.ParseNext() for template %q did not produce the expected nodes.\n", tt.template)
				t.Logf("Got (%d items):\n", len(got))
				for i, n := range got {
					t.Logf("  [%d] Type: %v, Content: %q\n", i, n.Type, n.Content)
				}
				t.Logf("Want (%d items):\n", len(tt.want))
				for i, n := range tt.want {
					t.Logf("  [%d] Type: %v, Content: %q\n", i, n.Type, n.Content)
				}
				t.FailNow()
			}
		})
	}
}

func TestParseNext_WithComplexInputs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []Node
	}{
		{
			name:  "text, expression, text",
			input: "hello {{ name }} world",
			want: []Node{
				{Type: NodeText, Content: "hello "},
				{Type: NodeExpression, Content: " name "},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:  "expression at start",
			input: "{{ greeting }} users",
			want: []Node{
				{Type: NodeExpression, Content: " greeting "},
				{Type: NodeText, Content: " users"},
			},
		},
		{
			name:  "expression at end",
			input: "count: {{ value }}",
			want: []Node{
				{Type: NodeText, Content: "count: "},
				{Type: NodeExpression, Content: " value "},
			},
		},
		{
			name:  "multiple expressions",
			input: "{{one}}{{ two }} {{three}}",
			want: []Node{
				{Type: NodeExpression, Content: "one"},
				{Type: NodeExpression, Content: " two "},
				{Type: NodeText, Content: " "},
				{Type: NodeExpression, Content: "three"},
			},
		},
		{
			name:  "empty input",
			input: "",
			want:  []Node{},
		},
		{
			name:  "only text",
			input: "this is just text.",
			want:  []Node{{Type: NodeText, Content: "this is just text."}},
		},
		{
			name:  "only expression",
			input: "{{alone}}",
			want:  []Node{{Type: NodeExpression, Content: "alone"}},
		},
		{
			name:  "unclosed expression",
			input: "hello {{ name",
			want:  []Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{{ name"}},
		},
		{
			name:  "unclosed expression with text after",
			input: "hello {{ name world",
			want:  []Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{{ name world"}},
		},
		{
			name:  "text with {{ literal",
			input: "hello {{ literal",
			want:  []Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{{ literal"}},
		},
		{
			name:  "expression with internal {{ but valid end",
			input: "{{ greeting }} {{ user_name }}",
			want: []Node{
				{Type: NodeExpression, Content: " greeting "},
				{Type: NodeText, Content: " "},
				{Type: NodeExpression, Content: " user_name "},
			},
		},
		{
			name:  "unclosed expression with internal {{ and text after",
			input: "hello {{ name {{ nested_var world",
			want:  []Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{{ name "}, {Type: NodeText, Content: "{{ nested_var world"}},
		},
		{
			name:  "expression containing escaped quotes and delimiters",
			input: "{{ a_var_with_\"string_literal\" }}",
			want: []Node{
				{Type: NodeExpression, Content: " a_var_with_\"string_literal\" "},
			},
		},
		{
			name:  "expression with quotes inside",
			input: "{{ task_result.stdout | from_json | map(attribute=\"name\") }}",
			want: []Node{
				{Type: NodeExpression, Content: " task_result.stdout | from_json | map(attribute=\"name\") "},
			},
		},
		{
			name:  "complex expression with nested structure-like syntax (not nested {{ and }} )",
			input: "{{ {'key': value, 'other': 'string'} }}",
			want: []Node{
				{Type: NodeExpression, Content: " {'key': value, 'other': 'string'} "},
			},
		},
		{
			name:  "text with '{' but not '{{'",
			input: "text { not an expression",
			want:  []Node{{Type: NodeText, Content: "text { not an expression"}},
		},
		{
			name:  "text with single '{' followed by '{{'",
			input: "text { {{ expr }}",
			want: []Node{
				{Type: NodeText, Content: "text { "},
				{Type: NodeExpression, Content: " expr "},
			},
		},
		{
			name:  "unclosed expression at EOF",
			input: "{{ unclosed",
			want:  []Node{{Type: NodeText, Content: "{{ unclosed"}},
		},
		{
			name:  "expression with only spaces",
			input: "{{   }}",
			want:  []Node{{Type: NodeExpression, Content: "   "}},
		},
		{
			name:  "text before unclosed expression",
			input: "leading text {{ var",
			want:  []Node{{Type: NodeText, Content: "leading text "}, {Type: NodeText, Content: "{{ var"}},
		},
		{
			name:  "text with {{ and }} but not an expression due to spacing",
			input: "text { {var} } world", // Invalid due to spaces, assuming strict {{ and }}
			want:  []Node{{Type: NodeText, Content: "text { {var} } world"}},
		},
		{
			name:  "expression like {{foo}}bar",
			input: "{{foo}}bar",
			want: []Node{
				{Type: NodeExpression, Content: "foo"},
				{Type: NodeText, Content: "bar"},
			},
		},
		{
			name:  "literal {{ appearing mid-text",
			input: "This is some text {{and this is an expression}} more text",
			want: []Node{
				{Type: NodeText, Content: "This is some text "},
				{Type: NodeExpression, Content: "and this is an expression"},
				{Type: NodeText, Content: " more text"},
			},
		},
		{
			name:  "text looks like start of expression but is not",
			input: "Text{Text",
			want:  []Node{{Type: NodeText, Content: "Text{Text"}},
		},
		{
			name:  "Text only containing {{ but not closed",
			input: "{{",
			want:  []Node{{Type: NodeText, Content: "{{"}},
		},
		{
			name:  "Text only containing { but not closed",
			input: "{",
			want:  []Node{{Type: NodeText, Content: "{"}},
		},
		{
			name:  "Text with {{var then text",
			input: "{{var then text",
			want:  []Node{{Type: NodeText, Content: "{{var then text"}},
		},
		{
			name:  "nested {{ and }} inside expression",
			input: "{{ outer_var + {{ inner_var }} }}", // This is tricky, Jinja might handle it or error
			want: []Node{
				{Type: NodeExpression, Content: " outer_var + {{ inner_var }} "}, // Current parser behavior
			},
		},
		{
			name:  "very short string - single curly brace",
			input: "{",
			want: []Node{
				{Type: NodeText, Content: "{"},
			},
		},
		{
			name:  "very short string - double curly brace open",
			input: "{{",
			want: []Node{
				{Type: NodeText, Content: "{{"}, // Treated as text because it's unclosed
			},
		},
		{
			name:  "expression with escaped quotes",
			input: "{{ \"alpha\\\"beta\\\\gamma\" }}", // Jinja: {{ "alpha"beta\gamma" }}
			want: []Node{
				{Type: NodeExpression, Content: " \"alpha\\\"beta\\\\gamma\" "},
			},
		},
		{
			name:  "text with unclosed expression followed by another expression",
			input: "text {{ unclosed {{ expr }}",
			want: []Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{{ unclosed "},
				{Type: NodeExpression, Content: " expr "},
			},
		},
		// Comment tests
		{
			name:  "simple comment",
			input: "{# this is a comment #}",
			want:  []Node{{Type: NodeComment, Content: " this is a comment "}},
		},
		{
			name:  "comment with text before and after",
			input: "hello {# comment #} world",
			want: []Node{
				{Type: NodeText, Content: "hello "},
				{Type: NodeComment, Content: " comment "},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:  "comment at start",
			input: "{# comment #} world",
			want: []Node{
				{Type: NodeComment, Content: " comment "},
				{Type: NodeText, Content: " world"},
			},
		},
		{
			name:  "comment at end",
			input: "hello {# comment #}",
			want: []Node{
				{Type: NodeText, Content: "hello "},
				{Type: NodeComment, Content: " comment "},
			},
		},
		{
			name:  "multiple comments",
			input: "{#c1#}{#c2#}",
			want: []Node{
				{Type: NodeComment, Content: "c1"},
				{Type: NodeComment, Content: "c2"},
			},
		},
		{
			name:  "comment containing expression-like syntax",
			input: "{# {{ not_an_expression }} #}",
			want:  []Node{{Type: NodeComment, Content: " {{ not_an_expression }} "}},
		},
		{
			name:  "unclosed comment",
			input: "hello {# comment world",
			want:  []Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{# comment world"}},
		},
		{
			name:  "unclosed comment at EOF",
			input: "{# unclosed",
			want:  []Node{{Type: NodeText, Content: "{# unclosed"}},
		},
		{
			name:  "text with {# literal",
			input: "hello {# literal",
			want:  []Node{{Type: NodeText, Content: "hello "}, {Type: NodeText, Content: "{# literal"}},
		},
		{
			name:  "text with {# and #} but not a comment due to spacing or content",
			input: "text { #var# } world", // Assuming strict {# and #}
			want:  []Node{{Type: NodeText, Content: "text { #var# } world"}},
		},
		{
			name:  "text only containing {# but not closed",
			input: "{#",
			want:  []Node{{Type: NodeText, Content: "{#"}},
		},
		{
			name:  "comment and expression mixed",
			input: "text {# comment #} {{ expr }} {# comment2 #}",
			want: []Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeComment, Content: " comment "},
				{Type: NodeText, Content: " "},
				{Type: NodeExpression, Content: " expr "},
				{Type: NodeText, Content: " "},
				{Type: NodeComment, Content: " comment2 "},
			},
		},
		{
			name:  "text, comment, expression, text, comment, expression",
			input: "A {# B #} C {{ D }} E {# F #} G {{ H }}",
			want: []Node{
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
			want: []Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{# unclosed "},
				{Type: NodeExpression, Content: " expr "},
			},
		},
		{
			name:  "unclosed expr followed by valid comment",
			input: "text {{ unclosed {# comment #}",
			want: []Node{
				{Type: NodeText, Content: "text "},
				{Type: NodeText, Content: "{{ unclosed "},
				{Type: NodeComment, Content: " comment "},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser(tt.input)
			var got []Node
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
				// Create a copy of the node to avoid issues if the parser reuses node memory (though it doesn't currently)
				// And to ensure we are comparing value semantics for 'want'
				nodeCopy := *node
				got = append(got, nodeCopy)
			}

			if (currentErr != nil) != false {
				t.Errorf("p.ParseNext() error = %v, wantErr %v\n", currentErr, false)
				return
			}

			// Handle empty 'want' specifically, as DeepEqual(got, nil) when got is empty slice is false.
			if len(tt.want) == 0 && len(got) == 0 {
				// This is a pass, both are effectively empty.
			} else if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("p.ParseNext() for input %q did not produce the expected nodes.\n", tt.input)
				t.Logf("Got (%d items):\n", len(got))
				for i, n := range got {
					t.Logf("  [%d] Type: %v, Content: %q\n", i, n.Type, n.Content)
				}
				t.Logf("Want (%d items):\n", len(tt.want))
				for i, n := range tt.want {
					t.Logf("  [%d] Type: %v, Content: %q\n", i, n.Type, n.Content)
				}
				t.FailNow()
			}
		})
	}
}
