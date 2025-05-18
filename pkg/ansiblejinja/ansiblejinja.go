package ansiblejinja

import (
	"fmt"
	"reflect"
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
				conditionResult, evalErr := EvaluateExpression(node.Control.Expression, context)
				if evalErr != nil {
					// Propagate errors from condition evaluation, e.g., undefined variable in condition
					return "", fmt.Errorf("error evaluating if-condition '%s': %v", node.Control.Expression, evalErr)
				}

				truthy, truthErr := isTruthy(conditionResult)
				if truthErr != nil {
					return "", fmt.Errorf("error determining truthiness of if-condition '%s': %v", node.Control.Expression, truthErr)
				}

				// Find the corresponding {% endif %}, potentially skipping {% else %} or {% elif %} for now.
				// findBlock needs to be aware of if/else/elif/endif structure.
				// For this iteration, findBlock will just look for a matching endif.
				blockNodes, endifIndex, blockErr := findBlock(nodes, currentIndex, ControlIf, ControlEndIf, ControlElse, ControlElseIf)
				if blockErr != nil {
					return "", blockErr // e.g., unclosed if statement
				}

				if truthy {
					// Process the nodes within the if block (nodes between if and endif/else/elif)
					blockContent, err := processNodes(blockNodes, context)
					if err != nil {
						return "", err
					}
					result.WriteString(blockContent)
				}
				// Move currentIndex past the entire if...endif block
				currentIndex = endifIndex + 1

			case ControlEndIf:
				// This should only be reached if findBlock logic is flawed or an endif is orphaned.
				return "", fmt.Errorf("template error: unexpected '{%% endif %%}' found at node index %d. Content: %s", currentIndex, node.Content)
			case ControlElse, ControlElseIf: // Placeholders for future
				return "", fmt.Errorf("control tag type '%s' not yet implemented", node.Control.Type)
			case ControlFor, ControlEndFor: // Placeholders for future
				return "", fmt.Errorf("control tag type '%s' not yet implemented", node.Control.Type)
			case ControlUnknown:
				return "", fmt.Errorf("unknown or malformed control tag '{%% %s %%}'", node.Content)
			default:
				return "", fmt.Errorf("unhandled control tag type in processNodes: %s", node.Control.Type)
			}
		default:
			return "", fmt.Errorf("unknown node type encountered during processing: %v", node.Type)
		}
	}
	return result.String(), nil
}

// findBlock locates the nodes within a control block (e.g., if...endif) and the index of the closing tag.
// It handles nested blocks of the same type (e.g., nested ifs).
// For if-blocks, it returns nodes between {% if %} and the next {% else %}, {% elif %}, or {% endif %}.
// The caller (processNodes) will then decide what to do based on the condition.
func findBlock(nodes []*Node, startIndex int, primaryStartType ControlTagType, primaryEndType ControlTagType, intermediateTypes ...ControlTagType) (blockNodes []*Node, closingTagIndex int, err error) {
	if startIndex >= len(nodes) || nodes[startIndex].Type != NodeControlTag || nodes[startIndex].Control == nil || nodes[startIndex].Control.Type != primaryStartType {
		return nil, -1, fmt.Errorf("internal error: findBlock called with incorrect start node at index %d for type %s", startIndex, primaryStartType)
	}

	nestingLevel := 1
	searchIndex := startIndex + 1

	for searchIndex < len(nodes) {
		node := nodes[searchIndex]
		if node.Type == NodeControlTag && node.Control != nil {
			switch node.Control.Type {
			case primaryStartType: // Nested block of the same kind, e.g., nested {% if %}
				nestingLevel++
			case primaryEndType: // Closing tag for the block, e.g., {% endif %}
				nestingLevel--
				if nestingLevel == 0 {
					// Found the matching end tag for the current block level.
					// The nodes for this segment of the block are from startIndex+1 to searchIndex-1.
					return nodes[startIndex+1 : searchIndex], searchIndex, nil
				}
			default:
				// Check for intermediate tags only at the current nesting level (level 1 for the outermost block)
				if nestingLevel == 1 {
					for _, intermediateType := range intermediateTypes {
						if node.Control.Type == intermediateType { // e.g. {% else %} or {% elif %}
							// Found an intermediate tag. The current block ends here.
							// The nodes for this segment are from startIndex+1 to searchIndex-1.
							return nodes[startIndex+1 : searchIndex], searchIndex, nil
						}
					}
				}
			}
		}
		searchIndex++
	}

	// If loop finishes, the block was not properly closed.
	return nil, -1, fmt.Errorf("unclosed '%s' tag starting at node index %d (content: '%s')", primaryStartType, startIndex, nodes[startIndex].Content)
}

// isTruthy determines the truthiness of a value according to Jinja rules.
// False: false, 0, empty string, empty list/map, nil.
// True: everything else.
func isTruthy(value interface{}) (bool, error) {
	if value == nil {
		return false, nil
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() != 0, nil
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0.0, nil // Important: compare to 0.0 for floats
	case reflect.String:
		return v.Len() > 0, nil
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan:
		// Check for nil first for map/slice/chan/ptr which can be nil and have Len panic or misreport.
		if v.IsNil() {
			return false, nil
		}
		return v.Len() > 0, nil
	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return false, nil
		}
		// For a non-nil pointer or interface, Jinja would look at the underlying value.
		// This requires a recursive call to isTruthy with the element.
		// If the element itself is nil (e.g. pointer to nil interface), it's false.
		return isTruthy(v.Elem().Interface()) // Recursively check the pointed-to value
	default:
		// By default, if it's not a known falsey type and not nil, consider it true.
		// Jinja is generally quite liberal with truthiness for custom objects.
		return true, nil // Or an error for unhandled types: fmt.Errorf("cannot determine truthiness of type %T", value)
	}
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
