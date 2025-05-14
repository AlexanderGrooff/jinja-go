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
