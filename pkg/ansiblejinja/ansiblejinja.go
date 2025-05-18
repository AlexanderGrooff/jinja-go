package ansiblejinja

import (
	"fmt"
	"strings"
)

// TemplateString renders a template string using the provided context.
// It processes Jinja-like expressions {{ ... }}, comments {# ... #}, and control tags {% ... %}.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	p := NewParser(template)
	var allNodes []*Node

	// First pass: Parse the entire template into a sequence of nodes.
	for {
		node, parseErr := p.ParseNext() // ParseNext now populates node.Control for control tags
		if parseErr != nil {
			// If ParseNext itself returns an error (e.g., for syntax errors it can detect)
			return "", fmt.Errorf("parsing error: %v", parseErr)
		}
		if node == nil { // EOF
			break
		}
		allNodes = append(allNodes, node)
	}

	// Second pass: Process the nodes, handling control flow.
	return processNodes(allNodes, context)
}

// processNodes recursively processes a slice of nodes, handling control flow like {% if %}.
func processNodes(nodes []*Node, context map[string]interface{}) (string, error) {
	var result strings.Builder
	currentIndex := 0

	for currentIndex < len(nodes) {
		node := nodes[currentIndex]

		switch node.Type {
		case NodeText:
			result.WriteString(node.Content)
			currentIndex++
		case NodeExpression:
			evalResult, wasUndefined, evalErr := evaluateFullExpressionInternal(node.Content, context)
			if evalErr != nil {
				if strings.Contains(evalErr.Error(), "nested {{ or }} found") || strings.Contains(evalErr.Error(), "unclosed expression tag") {
					return "", fmt.Errorf("error processing template expression: %w", evalErr)
				}
				if wasUndefined && evalResult == nil {
					// Append nothing for undefined variables not handled by a filter (like default)
				} else if evalErr != nil && !wasUndefined {
					return "", fmt.Errorf("error evaluating expression '%s': %v", node.Content, evalErr)
				} else {
					result.WriteString(fmt.Sprintf("%v", evalResult))
				}
			} else {
				if evalResult == nil && wasUndefined {
					// append nothing
				} else {
					result.WriteString(fmt.Sprintf("%v", evalResult))
				}
			}
			currentIndex++
		case NodeComment:
			// Comments are ignored
			currentIndex++
		case NodeControlTag:
			if node.Control == nil {
				return "", fmt.Errorf("internal parser error: NodeControlTag has nil Control info for content '%s'", node.Content)
			}
			switch node.Control.Type {
			case ControlIf:
				// Pass EvaluateExpression and processNodes as arguments to the handler
				renderedBlock, nextIdx, err := handleIfStatement(nodes, currentIndex, context, EvaluateExpression, processNodes)
				if err != nil {
					return "", err
				}
				result.WriteString(renderedBlock)
				currentIndex = nextIdx
			case ControlEndIf:
				// This should only be reached if findBlock logic is flawed or an endif is orphaned.
				return "", fmt.Errorf("template error: unexpected '{%% endif %%}' found at node index %d. Content: %s", currentIndex, node.Content)
			case ControlElse, ControlElseIf:
				return "", fmt.Errorf("template error: unexpected '{%% %s %%}' found outside of an if block at node index %d. Content: %s", node.Control.Type, currentIndex, node.Content)
			case ControlFor, ControlEndFor:
				return "", fmt.Errorf("control tag type '%s' not yet implemented", node.Control.Type)
			case ControlUnknown:
				// The parser stores the detailed parsing error in node.Control.Expression for ControlUnknown tags.
				return "", fmt.Errorf("unknown or malformed control tag '{%% %s %%}': %s", node.Content, node.Control.Expression)
			default:
				return "", fmt.Errorf("unhandled control tag type in processNodes: %s", node.Control.Type)
			}
		default:
			return "", fmt.Errorf("unknown node type encountered during processing: %v", node.Type)
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
