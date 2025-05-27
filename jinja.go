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

	// Special handling for expressions with dot notation and any operators
	if strings.Contains(trimmedExpression, ".") {
		result, err := evaluateExpressionWithDotNotation(trimmedExpression, context)
		if err == nil {
			return result, nil
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
	// Handle simple dot notation without any operators first
	if !strings.Contains(expr, " ") && strings.Contains(expr, ".") {
		return evaluateDotNotation(expr, context)
	}

	// Handle "not" prefix for dot notation
	if strings.HasPrefix(expr, "not ") {
		varName := strings.TrimSpace(strings.TrimPrefix(expr, "not "))
		if strings.Contains(varName, ".") && !strings.Contains(varName, " ") {
			val, err := evaluateDotNotation(varName, context)
			if err == nil && val != nil {
				return !IsTruthy(val), nil
			}
		}
	}

	// Use regex to find all dot notation variables in the expression
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

// ParseVariables extracts all Jinja variable names from a template string.
// It returns a slice of unique variable names found in expressions {{ ... }} and control tags {% ... %}.
// For example, "some string with a {{ item.name | default('name') }}" returns ["item"].
func ParseVariables(template string) ([]string, error) {
	// Parse the template into nodes
	parser := NewParser(template)
	nodes, err := parser.ParseAll()
	if err != nil {
		return nil, fmt.Errorf("template parsing error: %w", err)
	}

	// Use a map to track unique variable names
	variableSet := make(map[string]bool)

	// Extract variables from all nodes
	err = extractVariablesFromNodes(nodes, variableSet)
	if err != nil {
		return nil, fmt.Errorf("variable extraction error: %w", err)
	}

	// Convert map keys to slice
	variables := make([]string, 0, len(variableSet))
	for varName := range variableSet {
		variables = append(variables, varName)
	}

	return variables, nil
}

// extractVariablesFromNodes recursively extracts variable names from a slice of nodes
func extractVariablesFromNodes(nodes []*Node, variableSet map[string]bool) error {
	for _, node := range nodes {
		switch node.Type {
		case NodeExpression:
			// Extract variables from expression content
			err := extractVariablesFromExpression(node.Content, variableSet)
			if err != nil {
				return fmt.Errorf("error extracting variables from expression '{{ %s }}': %v", node.Content, err)
			}

		case NodeControlTag:
			if node.Control != nil {
				// Extract variables from control tag expressions
				switch node.Control.Type {
				case ControlIf, ControlElseIf:
					// Extract variables from if/elif condition
					if node.Control.Expression != "" {
						err := extractVariablesFromExpression(node.Control.Expression, variableSet)
						if err != nil {
							return fmt.Errorf("error extracting variables from control expression '%s': %v", node.Control.Expression, err)
						}
					}
				case ControlFor:
					// Extract variables from for loop expression
					if node.Control.Expression != "" {
						err := extractVariablesFromForExpression(node.Control.Expression, variableSet)
						if err != nil {
							return fmt.Errorf("error extracting variables from for expression '%s': %v", node.Control.Expression, err)
						}
					}
				}
			}

		case NodeText, NodeComment:
			// No variables to extract from text or comments
			continue
		}
	}
	return nil
}

// extractVariablesFromExpression extracts variable names from a Jinja expression string
func extractVariablesFromExpression(expression string, variableSet map[string]bool) error {
	trimmedExpr := strings.TrimSpace(expression)
	if trimmedExpr == "" {
		return nil
	}

	// Try to parse the expression using the LALR parser
	lexer := NewLexer(trimmedExpr)
	tokens, err := lexer.Tokenize()
	if err != nil {
		// If tokenization fails, fall back to simple regex-based extraction
		return extractVariablesWithRegex(trimmedExpr, variableSet)
	}

	// Parse tokens into AST
	parser := NewExprParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		// If parsing fails, fall back to simple regex-based extraction
		return extractVariablesWithRegex(trimmedExpr, variableSet)
	}

	// Extract variables from the AST
	extractVariablesFromAST(ast, variableSet)
	return nil
}

// extractVariablesFromAST recursively extracts variable names from an expression AST
func extractVariablesFromAST(node *ExprNode, variableSet map[string]bool) {
	if node == nil {
		return
	}

	switch node.Type {
	case NodeIdentifier:
		// This is a variable reference - add the root variable name
		variableSet[node.Identifier] = true

	case NodeAttribute:
		// For attribute access like "item.name", we want the root variable "item"
		if len(node.Children) > 0 {
			extractVariablesFromAST(node.Children[0], variableSet)
		}

	case NodeSubscript:
		// For subscript access like "item[0]", we want the root variable "item"
		if len(node.Children) > 0 {
			extractVariablesFromAST(node.Children[0], variableSet)
		}
		// Also check the subscript expression for variables
		if len(node.Children) > 1 {
			extractVariablesFromAST(node.Children[1], variableSet)
		}

	case NodeFunctionCall:
		// For function calls, extract variables from arguments only, not the function name
		// The first child is the function name/identifier, skip it
		// Extract variables from arguments (children[1:])
		for i := 1; i < len(node.Children); i++ {
			extractVariablesFromAST(node.Children[i], variableSet)
		}

	case NodeUnaryOp, NodeBinaryOp:
		// For operators, extract variables from all operands
		for _, child := range node.Children {
			extractVariablesFromAST(child, variableSet)
		}

	case NodeList, NodeDict, NodeTuple:
		// For collections, extract variables from all elements
		for _, child := range node.Children {
			extractVariablesFromAST(child, variableSet)
		}

	case NodeLiteral:
		// Literals don't contain variables
		return
	}
}

// extractVariablesFromForExpression extracts variables from a for loop expression like "item in items"
func extractVariablesFromForExpression(expression string, variableSet map[string]bool) error {
	trimmedExpr := strings.TrimSpace(expression)

	// For expressions have the format: "variable in iterable" or "key, value in dict"
	// We want to extract the iterable part, not the loop variables
	inIndex := strings.Index(trimmedExpr, " in ")
	if inIndex == -1 {
		// Invalid for expression, but try to extract any variables anyway
		return extractVariablesFromExpression(trimmedExpr, variableSet)
	}

	// Extract the iterable part (after " in ")
	iterablePart := strings.TrimSpace(trimmedExpr[inIndex+4:])
	return extractVariablesFromExpression(iterablePart, variableSet)
}

// extractVariablesWithRegex is a fallback method that uses regex to extract variable names
// when the LALR parser fails
func extractVariablesWithRegex(expression string, variableSet map[string]bool) error {
	// Remove filter expressions (everything after |)
	if pipeIndex := strings.Index(expression, "|"); pipeIndex != -1 {
		expression = strings.TrimSpace(expression[:pipeIndex])
	}

	// Simple regex to match identifiers (variable names)
	// This matches sequences of letters, digits, and underscores that start with a letter or underscore
	identifierPattern := regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*\b`)
	matches := identifierPattern.FindAllString(expression, -1)

	for _, match := range matches {
		// Skip common keywords and literals
		switch match {
		case "True", "False", "None", "true", "false", "none", "null",
			"and", "or", "not", "in", "is", "if", "else", "elif", "for", "endfor", "endif":
			continue
		default:
			// Extract the root variable name (before any dots)
			if dotIndex := strings.Index(match, "."); dotIndex != -1 {
				match = match[:dotIndex]
			}
			variableSet[match] = true
		}
	}

	return nil
}
