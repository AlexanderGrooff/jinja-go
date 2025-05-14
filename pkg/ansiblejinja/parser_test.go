package ansiblejinja

import (
	"reflect"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	tests := []struct {
		name     string
		template string
		want     []segment
	}{
		{
			name:     "empty string",
			template: "",
			want:     nil,
		},
		{
			name:     "only literal text",
			template: "Hello, world!",
			want:     []segment{{literalText, "Hello, world!"}},
		},
		{
			name:     "simple expression",
			template: "{{ name }}",
			want:     []segment{{expressionTag, " name "}},
		},
		{
			name:     "literal then expression",
			template: "Hello {{ name }}",
			want: []segment{
				{literalText, "Hello "},
				{expressionTag, " name "},
			},
		},
		{
			name:     "expression then literal",
			template: "{{ name }}!",
			want: []segment{
				{expressionTag, " name "},
				{literalText, "!"},
			},
		},
		{
			name:     "literal, expression, literal",
			template: "Hello {{ name }}!",
			want: []segment{
				{literalText, "Hello "},
				{expressionTag, " name "},
				{literalText, "!"},
			},
		},
		{
			name:     "multiple expressions",
			template: "{{ greeting }} {{ name }}!",
			want: []segment{
				{expressionTag, " greeting "},
				{literalText, " "},
				{expressionTag, " name "},
				{literalText, "!"},
			},
		},
		{
			name:     "consecutive expressions",
			template: "{{first}}{{second}}",
			want: []segment{
				{expressionTag, "first"},
				{expressionTag, "second"},
			},
		},
		{
			name:     "unclosed tag at end of string",
			template: "Hello {{",
			want:     []segment{{literalText, "Hello {{"}}, // Treats unclosed as literal
		},
		{
			name:     "unclosed tag with content at end of string",
			template: "Hello {{ name",
			want:     []segment{{literalText, "Hello {{ name"}},
		},
		{
			name:     "unclosed tag in middle of string",
			template: "Hello {{ name and goodbye",
			want:     []segment{{literalText, "Hello {{ name and goodbye"}},
		},
		{
			name:     "unclosed tag followed by another literal",
			template: "text {{ unclosed text_after",
			want:     []segment{{literalText, "text {{ unclosed text_after"}},
		},
		{
			name:     "double open brace literal, not a tag",
			template: "This has { { not a tag } }",
			want:     []segment{{literalText, "This has { { not a tag } }"}},
		},
		{
			name:     "expression with no spaces",
			template: "{{name}}",
			want:     []segment{{expressionTag, "name"}},
		},
		{
			name:     "expression with leading space",
			template: "{{ name}}",
			want:     []segment{{expressionTag, " name"}},
		},
		{
			name:     "expression with trailing space",
			template: "{{name }}",
			want:     []segment{{expressionTag, "name "}},
		},
		{
			name:     "empty expression",
			template: "{{}}",
			want:     []segment{{expressionTag, ""}},
		},
		{
			name:     "expression with only spaces",
			template: "{{   }}",
			want:     []segment{{expressionTag, "   "}},
		},
		{
			name:     "string with only {{ }} then text",
			template: "{{}} world",
			want: []segment{
				{expressionTag, ""},
				{literalText, " world"},
			},
		},
		{
			name:     "text then {{}} then text",
			template: "hello{{}}world",
			want: []segment{
				{literalText, "hello"},
				{expressionTag, ""},
				{literalText, "world"},
			},
		},
		{
			name:     "incomplete open tag at very end {{ a",
			template: "test {{ a",
			want:     []segment{{literalText, "test {{ a"}},
		},
		{
			name:     "incomplete open tag at very end {",
			template: "test {",
			want:     []segment{{literalText, "test {"}},
		},
		{
			name:     "template starting with unclosed tag",
			template: "{{ unfinished then text",
			want:     []segment{{literalText, "{{ unfinished then text"}},
		},
		{
			name:     "template with {{ and }} but content missing",
			template: "{{}} and {{ value }}",
			want: []segment{
				{expressionTag, ""},
				{literalText, " and "},
				{expressionTag, " value "},
			},
		},
		{
			name:     "only an unclosed tag {{tag",
			template: "{{tag",
			want:     []segment{{literalText, "{{tag"}},
		},
		{
			name:     "only {{ at end",
			template: "text{{",
			want:     []segment{{literalText, "text{{"}},
		},
		{
			name:     "string expression inside expression",
			template: "{{ 'some string' }}",
			want:     []segment{{expressionTag, " 'some string' "}},
		},
		{
			name:     "expression opening braces inside expression",
			template: "{{ '{{' }}",
			want:     []segment{{expressionTag, " '{{' "}},
		},
		{
			name:     "expression closing braces inside expression",
			template: "{{ '}}' }}",
			want:     []segment{{expressionTag, " '}}' "}},
		},
		{
			name:     "string expression with escaped braces inside expression",
			template: "{{ '{{}}' }}",
			want:     []segment{{expressionTag, " '{{}}' "}},
		},
		{
			name:     "string expression with escaped double quotes inside expression",
			template: "{{ '\"' }}",
			want:     []segment{{expressionTag, " '\"' "}},
		},
		{
			name:     "escaped nested expression",
			template: "{{ \"{{ name }}\" }}",
			want:     []segment{{expressionTag, " \"{{ name }}\" "}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTemplate(tt.template)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTemplate(%q) = %v, want %v", tt.template, got, tt.want)
			}
		})
	}
}
