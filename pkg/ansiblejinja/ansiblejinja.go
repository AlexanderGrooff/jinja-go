package ansiblejinja

import (
	"bytes"
	"fmt"
	"strings"
)

// TemplateString renders a template string using the provided context.
// It processes Jinja-like expressions {{ ... }} and applies filters.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	segments := parseTemplate(template) // From parser.go
	var result bytes.Buffer

	for _, seg := range segments {
		if seg.segmentType == literalText { // literalText is from parser.go
			result.WriteString(seg.content)
		} else if seg.segmentType == expressionTag { // expressionTag is from parser.go
			// Check for nested expressions, which are not allowed by Ansible and should error.
			// This check might be more robustly handled directly in the parser if it builds an AST
			// or if parseTemplate explicitly errors on them.
			// For now, a simpler check on the content of an expressionTag segment.
			if strings.Contains(seg.content, "{{") || strings.Contains(seg.content, "}}") {
				// This check is simplistic. parseTemplate's internal logic for balancing might
				// mean that by the time we get seg.content, it's already correctly bounded.
				// The error from Ansible is often when rendering {{ outer_var_renders_to_inner_tag }}
				// This specific case is hard to catch here without full recursive evaluation first.
				// The previous fix was to make parseTemplate handle nesting correctly to define segment boundaries.
				// If seg.content *still* contains {{ or }}, it implies an issue not caught by the primary parser loop or an unescaped literal.
				// Let's rely on evaluateFullExpressionInternal to catch syntax errors within the expression content.
			}

			evalResult, wasUndefined, err := evaluateFullExpressionInternal(seg.content, context) // from parser.go
			if err != nil {
				// According to the ansible error test case, it should return an empty string and an error.
				// For "var1 {{ var1 }} and var2 {{ {{ var2 }} }}", if `evaluateFullExpressionInternal` errors on `{{ var2 }}`,
				// this matches the desired behavior.
				return "", fmt.Errorf("failed to evaluate expression '%s': %v", seg.content, err)
			}
			if wasUndefined {
				// If strictly undefined and not handled by a filter (like default), Jinja typically renders an empty string.
				// result.WriteString("") // No need to write empty string
			} else if evalResult != nil {
				result.WriteString(fmt.Sprint(evalResult))
			} else {
				// evalResult is nil, but wasUndefined is false. This can happen if a variable resolves to nil
				// or a filter returns nil explicitly. Jinja often renders this as an empty string.
				// Example: context{"foo": nil}, template "{{ foo }}" -> result: ""
				// result.WriteString("") // No need to write empty string
			}
		} else {
			// Unknown segment type, should not happen with current parseTemplate
			return "", fmt.Errorf("unknown segment type: %v", seg.segmentType)
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
