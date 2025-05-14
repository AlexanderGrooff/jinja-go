package ansiblejinja

import (
	"fmt"
	"strconv"
	"strings"
)

// TemplateString evaluates a Jinja template string with the given context.
// It replaces expressions like {{ variable }} with their values from the context.
// If a variable is not found or the expression is not a recognized literal,
// it's replaced with an empty string.
// Nested variable constructs like {{ {{ variable }} }} will result in an error.
// If '{{' is not properly closed with '}}', it's treated as literal text.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	segments := parseTemplate(template) // Use the new parser
	var result strings.Builder

	for _, seg := range segments {
		switch seg.segmentType {
		case literalText:
			result.WriteString(seg.content)
		case expressionTag:
			rawContent := seg.content
			expressionKey := strings.TrimSpace(rawContent)

			// Check for Ansible-like nested variable error FIRST.
			// This error is about the *syntax* of the expression content.
			if strings.HasPrefix(expressionKey, "{{") && strings.HasSuffix(expressionKey, "}}") {
				// Check if it's a string literal that happens to contain "{{...}}"
				// If expressionKey is like ""{{ foo }}"", strconv.Unquote will succeed.
				// In that case, it's a string literal, not a nested variable error.
				_, err := strconv.Unquote(expressionKey)
				if err != nil { // If Unquote fails, it's not a valid string literal, so it IS a nested var error.
					return "", fmt.Errorf("template error while templating string: nested variable constructs like '{{ %s }}' are not supported directly. Original segment content: %s", expressionKey, rawContent)
				}
				// If Unquote succeeded, it's a string literal like "{{ "{{val}}" }}", proceed to EvaluateExpression.
			}

			evaluatedValue, evalErr := EvaluateExpression(expressionKey, context)
			if evalErr == nil {
				result.WriteString(fmt.Sprintf("%v", evaluatedValue))
			} else {
				// For TemplateString, if EvaluateExpression fails (e.g., variable not found
				// and not a recognized literal), it defaults to an empty string for that segment.
				// No error is propagated from TemplateString itself for this specific case.
			}
		}
	}

	return result.String(), nil
}

// EvaluateExpression evaluates a Jinja expression string with the given context.
// The expression can be a variable name or a string literal (e.g., "'hello'" or "\"world\"").
// If it's a string literal, its unquoted value is returned.
// Otherwise, it attempts to find the variable in the context.
// If the expression cannot be evaluated, an error is raised.
func EvaluateExpression(expression string, context map[string]interface{}) (interface{}, error) {
	cleanExpression := strings.TrimSpace(expression)

	// Try to interpret as a string literal first
	if unquoted, err := strconv.Unquote(cleanExpression); err == nil {
		return unquoted, nil
	}

	// If not a string literal, try context lookup
	if value, ok := context[cleanExpression]; ok {
		return value, nil
	}

	return nil, fmt.Errorf("failed to evaluate expression: '%s' is not a recognized literal and not found in context", cleanExpression)
}
