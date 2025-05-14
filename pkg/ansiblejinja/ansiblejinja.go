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
				return "", fmt.Errorf("failed to evaluate expression '%s': %v", node.Content, evalErr)
			}
			if wasUndefined {
				// If strictly undefined and not handled by a filter (like default), Jinja typically renders an empty string.
				// No action needed as we don't write anything to the result.
			} else if evalResult != nil {
				result.WriteString(fmt.Sprint(evalResult))
			} else {
				// evalResult is nil, but wasUndefined is false. This can happen if a variable resolves to nil
				// or a filter returns nil explicitly. Jinja often renders this as an empty string.
				// No action needed.
			}
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
