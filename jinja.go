package jinja

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

// TemplateCache is a thread-safe cache for parsed templates
type TemplateCache struct {
	cache map[string][]*Node
	mu    sync.RWMutex
}

// NewTemplateCache creates a new template cache
func NewTemplateCache() *TemplateCache {
	return &TemplateCache{
		cache: make(map[string][]*Node),
	}
}

// Get retrieves parsed nodes for a template from the cache
func (tc *TemplateCache) Get(template string) ([]*Node, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	nodes, ok := tc.cache[template]
	return nodes, ok
}

// Set stores parsed nodes for a template in the cache
func (tc *TemplateCache) Set(template string, nodes []*Node) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[template] = nodes
}

// Global template cache
var defaultTemplateCache = NewTemplateCache()

// TemplateString renders a template string using the provided context.
// It processes Jinja-like expressions {{ ... }}, comments {# ... #}, and control tags {% ... %}.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	// Check if this template is already cached
	nodes, found := defaultTemplateCache.Get(template)
	if !found {
		// Parse the template
		parser := NewParser(template)
		var err error
		nodes, err = parser.ParseAll()
		if err != nil {
			return "", fmt.Errorf("template parsing error: %w", err)
		}
		// Cache the parsed nodes
		defaultTemplateCache.Set(template, nodes)
	}

	// Render the template
	var sb strings.Builder

	// Handle control flow (if, for, etc.)
	err := renderNodes(nodes, context, &sb)
	if err != nil {
		return "", fmt.Errorf("template rendering error: %w", err)
	}

	return sb.String(), nil
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
			trimmedExpr := strings.TrimSpace(node.Content)

			// First try to handle special functions that require different evaluation
			if strings.HasPrefix(trimmedExpr, "lookup(") || strings.Contains(trimmedExpr, " lookup(") {
				// Handle lookup function specially
				val, err := ParseAndEvaluate(trimmedExpr, context)
				if err == nil {
					// Success! Convert the result to string and add to output
					switch v := val.(type) {
					case string:
						result.WriteString(v)
					case nil:
						// nil values render as empty strings
						// Do nothing, no output
					default:
						// For all other types, use fmt.Sprintf to get a string representation
						result.WriteString(fmt.Sprintf("%v", v))
					}
					currentIndex++
					continue
				} else {
					// For lookup errors, return a more specific error
					return "", fmt.Errorf("error in lookup function: %v", err)
				}
			}

			// For normal expressions, use the filter pipeline
			val, wasUndefined, err := evaluateFullExpressionInternal(node.Content, context)
			if err != nil {
				return "", fmt.Errorf("error evaluating expression '{{ %s }}': %v", node.Content, err)
			}

			if wasUndefined && val == nil {
				// Jinja2 renders undefined variables as empty strings
				currentIndex++
				continue
			}

			switch v := val.(type) {
			case string:
				result.WriteString(v)
			case nil:
				// nil values render as empty strings
				// Do nothing, no output
			default:
				// For all other types, use fmt.Sprintf to get a string representation
				result.WriteString(fmt.Sprintf("%v", v))
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
			case ControlFor:
				// Handle for loop
				renderedBlock, nextIdx, err := handleForStatement(nodes, currentIndex, context, EvaluateExpression, processNodes)
				if err != nil {
					return "", err
				}
				result.WriteString(renderedBlock)
				currentIndex = nextIdx
			case ControlEndFor:
				// This should only be reached if findBlock logic is flawed or an endfor is orphaned.
				return "", fmt.Errorf("template error: unexpected '{%% endfor %%}' found at node index %d. Content: %s", currentIndex, node.Content)
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

	// If the expression contains a filter pipe, prioritize the original filter pipeline
	if strings.Contains(trimmedExpression, "|") {
		val, wasStrictlyUndefined, err := evaluateFullExpressionInternal(trimmedExpression, context)
		if err == nil {
			if wasStrictlyUndefined {
				return nil, fmt.Errorf("variable '%s' is undefined", expression)
			}
			return val, nil
		}
	}

	// First try with the LALR parser for complex expressions
	// If the expression contains operators, attributes, or is a compound expression, this will handle it
	val, err := ParseAndEvaluate(trimmedExpression, context)
	if err == nil {
		return val, nil
	}

	// Special handling for expressions with complex dot notation
	// Look for patterns of dot notation with operators or parentheses
	if strings.Contains(trimmedExpression, ".") &&
		(strings.Contains(trimmedExpression, "(") ||
			strings.Contains(trimmedExpression, " + ") ||
			strings.Contains(trimmedExpression, " - ") ||
			strings.Contains(trimmedExpression, " * ") ||
			strings.Contains(trimmedExpression, " / ") ||
			strings.Contains(trimmedExpression, " % ") ||
			strings.Contains(trimmedExpression, " and ") ||
			strings.Contains(trimmedExpression, " or ")) {

		result, err := evaluateExpressionWithDotNotation(trimmedExpression, context)
		if err == nil {
			return result, nil
		}
	}

	// Special handling for expressions with dot notation and comparison operators
	if strings.Contains(trimmedExpression, ".") {
		// Handle expressions like "loop.index > 1"
		if strings.Contains(trimmedExpression, " > ") {
			parts := strings.Split(trimmedExpression, " > ")
			if len(parts) == 2 {
				leftVar := strings.TrimSpace(parts[0])
				rightVal := strings.TrimSpace(parts[1])

				// If left part contains dot notation, evaluate it first
				if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
					left, err := evaluateDotNotation(leftVar, context)
					if err == nil && left != nil {
						// Now create an expression like "2 > 1" that can be evaluated
						newExpr := fmt.Sprintf("%v > %s", left, rightVal)
						result, err := ParseAndEvaluate(newExpr, context)
						if err == nil {
							return result, nil
						}
					}
				}
			}
		}

		// Handle expressions like "loop.index <= 1"
		if strings.Contains(trimmedExpression, " <= ") {
			parts := strings.Split(trimmedExpression, " <= ")
			if len(parts) == 2 {
				leftVar := strings.TrimSpace(parts[0])
				rightVal := strings.TrimSpace(parts[1])

				// If left part contains dot notation, evaluate it first
				if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
					left, err := evaluateDotNotation(leftVar, context)
					if err == nil && left != nil {
						// Now create an expression like "2 <= 1" that can be evaluated
						newExpr := fmt.Sprintf("%v <= %s", left, rightVal)
						result, err := ParseAndEvaluate(newExpr, context)
						if err == nil {
							return result, nil
						}
					}
				}
			}
		}

		// Handle expressions like "loop.index < 1"
		if strings.Contains(trimmedExpression, " < ") {
			parts := strings.Split(trimmedExpression, " < ")
			if len(parts) == 2 {
				leftVar := strings.TrimSpace(parts[0])
				rightVal := strings.TrimSpace(parts[1])

				// If left part contains dot notation, evaluate it first
				if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
					left, err := evaluateDotNotation(leftVar, context)
					if err == nil && left != nil {
						// Now create an expression like "2 < 1" that can be evaluated
						newExpr := fmt.Sprintf("%v < %s", left, rightVal)
						result, err := ParseAndEvaluate(newExpr, context)
						if err == nil {
							return result, nil
						}
					}
				}
			}
		}

		// Handle expressions like "loop.index >= 1"
		if strings.Contains(trimmedExpression, " >= ") {
			parts := strings.Split(trimmedExpression, " >= ")
			if len(parts) == 2 {
				leftVar := strings.TrimSpace(parts[0])
				rightVal := strings.TrimSpace(parts[1])

				// If left part contains dot notation, evaluate it first
				if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
					left, err := evaluateDotNotation(leftVar, context)
					if err == nil && left != nil {
						// Now create an expression like "2 >= 1" that can be evaluated
						newExpr := fmt.Sprintf("%v >= %s", left, rightVal)
						result, err := ParseAndEvaluate(newExpr, context)
						if err == nil {
							return result, nil
						}
					}
				}
			}
		}

		// Handle expressions like "loop.index == 1"
		if strings.Contains(trimmedExpression, " == ") {
			parts := strings.Split(trimmedExpression, " == ")
			if len(parts) == 2 {
				leftVar := strings.TrimSpace(parts[0])
				rightVal := strings.TrimSpace(parts[1])

				// If left part contains dot notation, evaluate it first
				if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
					left, err := evaluateDotNotation(leftVar, context)
					if err == nil && left != nil {
						// Now create an expression like "2 == 1" that can be evaluated
						newExpr := fmt.Sprintf("%v == %s", left, rightVal)
						result, err := ParseAndEvaluate(newExpr, context)
						if err == nil {
							return result, nil
						}
					}
				}
			}
		}

		// Handle expressions like "loop.index != 1"
		if strings.Contains(trimmedExpression, " != ") {
			parts := strings.Split(trimmedExpression, " != ")
			if len(parts) == 2 {
				leftVar := strings.TrimSpace(parts[0])
				rightVal := strings.TrimSpace(parts[1])

				// If left part contains dot notation, evaluate it first
				if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
					left, err := evaluateDotNotation(leftVar, context)
					if err == nil && left != nil {
						// Now create an expression like "2 != 1" that can be evaluated
						newExpr := fmt.Sprintf("%v != %s", left, rightVal)
						result, err := ParseAndEvaluate(newExpr, context)
						if err == nil {
							return result, nil
						}
					}
				}
			}
		}

		// Handle expressions like "not loop.last"
		if strings.HasPrefix(trimmedExpression, "not ") {
			varName := strings.TrimSpace(strings.TrimPrefix(trimmedExpression, "not "))
			// Check if it's directly a dotted variable with no other operations
			if strings.Contains(varName, ".") && !strings.Contains(varName, " ") {
				// Evaluate the variable
				val, err := evaluateDotNotation(varName, context)
				if err == nil && val != nil {
					// Apply 'not' operator to the result
					return !IsTruthy(val), nil
				}
			}
		}

		// If it's just a simple dotted path like "user.name" with no operators
		if !strings.Contains(trimmedExpression, " ") {
			val, err := evaluateDotNotation(trimmedExpression, context)
			if err == nil {
				return val, nil
			}
		}
	}

	// If the expression contains attribute access or is complex, try ParseAndEvaluate next
	if strings.Contains(trimmedExpression, ".") ||
		strings.Contains(trimmedExpression, "[") ||
		strings.Contains(trimmedExpression, ">") ||
		strings.Contains(trimmedExpression, "<") ||
		strings.Contains(trimmedExpression, "!") ||
		strings.Contains(trimmedExpression, "==") ||
		strings.Contains(trimmedExpression, " not ") {
		val, err = ParseAndEvaluate(trimmedExpression, context)
		if err == nil {
			return val, nil
		}
	}

	// Fall back to the old evaluator for simpler expressions with filters
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

// For more complex expressions with dot notation, we can do a substitution approach:
// 1. Find all dot notation variables in the expression
// 2. Replace them with their evaluated values
// 3. Then evaluate the resulting expression with ParseAndEvaluate
func evaluateExpressionWithDotNotation(expr string, context map[string]interface{}) (interface{}, error) {
	// This is a simplified implementation
	// For a more robust solution, we would need to parse the expression properly

	// Check for simple dot notation in parentheses patterns like "(loop.index - 1)"
	dotNotationPattern := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*\.[a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := dotNotationPattern.FindAllString(expr, -1)

	// If we found dot notation variables, replace them with their values
	if len(matches) > 0 {
		tempExpr := expr
		for _, match := range matches {
			// Evaluate the dot notation
			val, err := evaluateDotNotation(match, context)
			if err != nil {
				return nil, fmt.Errorf("error evaluating dot notation '%s': %v", match, err)
			}

			// Replace the variable with its literal value
			// This is a simplistic approach that doesn't handle all edge cases
			var replacement string
			switch v := val.(type) {
			case string:
				replacement = fmt.Sprintf("'%s'", v)
			default:
				replacement = fmt.Sprintf("%v", v)
			}

			// Use regex to ensure we only replace the exact matches
			tempExpr = regexp.MustCompile(regexp.QuoteMeta(match)).ReplaceAllString(tempExpr, replacement)
		}

		// Now evaluate the expression with literals instead of dot notation
		return ParseAndEvaluate(tempExpr, context)
	}

	// If no dot notation was found, just evaluate normally
	return ParseAndEvaluate(expr, context)
}

// ParseAll parses the entire template into a slice of nodes.
func (p *Parser) ParseAll() ([]*Node, error) {
	var nodes []*Node
	for {
		node, err := p.ParseNext()
		if err != nil {
			return nil, err
		}
		if node == nil {
			break // End of template
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// renderNodes processes a slice of nodes and writes the result to the given strings.Builder.
func renderNodes(nodes []*Node, context map[string]interface{}, sb *strings.Builder) error {
	result, err := processNodes(nodes, context)
	if err != nil {
		return err
	}

	sb.WriteString(result)
	return nil
}
