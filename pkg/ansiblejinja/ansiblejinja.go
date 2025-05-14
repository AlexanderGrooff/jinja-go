package ansiblejinja

import (
	"fmt"
	"strings"
)

// TemplateString evaluates a Jinja template string with the given context.
// It replaces expressions like {{ variable }} with their values from the context.
// If a variable is not found, it's replaced with an empty string.
// If '{{' is not properly closed with '}}', it's treated as literal text.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	segments := parseTemplate(template) // Use the new parser
	var result strings.Builder

	for _, seg := range segments {
		switch seg.segmentType {
		case literalText:
			result.WriteString(seg.content)
		case expressionTag:
			// Extract and trim the expression key from the segment's content
			rawContent := seg.content
			expressionKey := strings.TrimSpace(rawContent)

			// Check for Ansible-like nested variable error
			if strings.HasPrefix(expressionKey, "{{") && strings.HasSuffix(expressionKey, "}}") {
				return "", fmt.Errorf("template error while templating string: nested variable constructs like '{{ %s }}' are not supported directly. Original segment content: %s", expressionKey, rawContent)
			}

			if value, ok := context[expressionKey]; ok {
				result.WriteString(fmt.Sprintf("%v", value))
			} else {
				// Variable not found in context, append empty string.
				// This matches the previous behavior. Stricter error handling
				// or different default behavior could be implemented here if needed.
			}
		}
	}

	return result.String(), nil
}

// EvaluateExpression evaluates a Jinja expression string with the given context.
// The expression is expected to be a simple variable name.
// If the expression cannot be evaluated (e.g., variable not in context), an error is raised.
func EvaluateExpression(expression string, context map[string]interface{}) (interface{}, error) {
	// Trim whitespace from the expression key
	cleanExpression := strings.TrimSpace(expression)

	if value, ok := context[cleanExpression]; ok {
		return value, nil
	}

	return nil, fmt.Errorf("variable '%s' not found in context", cleanExpression)
}
