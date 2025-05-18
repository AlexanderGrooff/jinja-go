package ansiblejinja

import (
	"fmt"
	"strings"
)

// TemplateString renders a template string using the provided context.
// It processes Jinja-like expressions {{ ... }} and applies filters.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	p := NewParser(template)
	var result strings.Builder

	for {
		node, err := p.ParseNext()
		if err != nil {
			// ParseNext currently doesn't return errors, but good to have for the future
			return "", fmt.Errorf("parsing error: %v", err)
		}
		if node == nil { // EOF
			break
		}

		switch node.Type {
		case NodeText:
			result.WriteString(node.Content)
		case NodeExpression:
			// node.Content is the raw string between {{ and }}
			// evaluateFullExpressionInternal will handle parsing this content, including filters.
			evalResult, wasUndefined, evalErr := evaluateFullExpressionInternal(node.Content, context)
			if evalErr != nil {
				// Check for specific error related to nested expressions
				if strings.Contains(evalErr.Error(), "nested {{ or }} found") || strings.Contains(evalErr.Error(), "unclosed expression tag") {
					return "", fmt.Errorf("error processing template: %w", evalErr)
				}
				// For TemplateString, if a simple variable is not found (strictlyUndefined is true) and no default filter resolved it,
				// Jinja typically renders it as an empty string rather than erroring out.
				// If an error occurred that isn't just a strict undefined lookup (e.g. bad filter syntax), that's different.
				if wasUndefined && evalResult == nil { // val might be non-nil if default filter handled it
					// Append nothing for undefined variables, effectively an empty string.
				} else if evalErr != nil && !wasUndefined { // A real error not from simple undefinedness
					return "", fmt.Errorf("error evaluating expression '%s': %v", node.Content, evalErr)
				} else {
					// It was strictly undefined but a filter (like default) might have provided a value.
					// Or it was found, or it was a literal.
					result.WriteString(fmt.Sprintf("%v", evalResult))
				}
			} else {
				// No error, evalResult contains the evaluated result (could be empty string for undefined if not handled by default)
				// or the actual value.
				if evalResult == nil && wasUndefined { // Explicitly handle if evaluate decided undefined means nil here
					// append nothing
				} else {
					result.WriteString(fmt.Sprintf("%v", evalResult))
				}
			}
		case NodeComment:
			// Comments are ignored, do nothing
		default:
			return "", fmt.Errorf("unknown node type: %v", node.Type)
		}
	}
	return result.String(), nil
}

// EvaluateExpression evaluates a single expression string (without surrounding {{ }})
// against the provided context. It applies filters as specified.
// If the variable is undefined after evaluation (and not handled by a filter like default),
// an error is returned.
func EvaluateExpression(expression string, context map[string]interface{}) (interface{}, error) {
	// Trim leading/trailing spaces from the raw expression string as the parser/evaluator expects clean content.
	trimmedExpression := strings.TrimSpace(expression)

	val, wasStrictlyUndefined, err := evaluateFullExpressionInternal(trimmedExpression, context) // from parser.go
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %v", expression, err)
	}

	if wasStrictlyUndefined {
		// For EvaluateExpression, strictly undefined (and not resolved by a filter like default)
		// should be an error, as per the project requirements.
		return nil, fmt.Errorf("variable '%s' is undefined", expression) // Or more specific part that was undefined
	}

	return val, nil
}
