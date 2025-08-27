package jinja

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
)

func init() {
	evaluateExpressionFunc = EvaluateExpression
}

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

// FileResolver resolves file contents for include/lookup with custom search rules
type FileResolver func(path string) ([]byte, error)

func defaultFileResolver(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// TemplateString renders a template string using the provided context.
// It processes Jinja-like expressions {{ ... }}, comments {# ... #}, and control tags {% ... %}.
func TemplateString(template string, context map[string]interface{}) (string, error) {
	return TemplateStringInContext(template, context, nil)
}

// TemplateStringInContext renders a template with custom directories used for file resolution.
// When searchDirs is non-empty, relative paths are resolved by checking each directory in order.
func TemplateStringInContext(template string, context map[string]interface{}, searchDirs []string) (string, error) {
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

	// Build resolver
	var resolver FileResolver = defaultFileResolver
	if len(searchDirs) > 0 {
		dirsCopy := append([]string(nil), searchDirs...)
		resolver = func(path string) ([]byte, error) {
			// Absolute path
			if strings.HasPrefix(path, "/") || (len(path) > 1 && path[1] == ':') {
				return os.ReadFile(path)
			}
			for _, d := range dirsCopy {
				if d == "" {
					continue
				}
				candidate := d
				if !strings.HasSuffix(candidate, "/") {
					candidate += "/"
				}
				candidate += path
				if b, err := os.ReadFile(candidate); err == nil {
					return b, nil
				}
			}
			return os.ReadFile(path)
		}
	}

	// Handle control flow (if, for, etc.)
	err := renderNodesWithResolver(nodes, context, &sb, resolver)
	if err != nil {
		return "", fmt.Errorf("template rendering error: %w", err)
	}

	return sb.String(), nil
}

// TryEvaluateSingleExpressionTemplate determines if the provided template consists of
// exactly one expression node (ignoring surrounding whitespace and comments), and if so,
// evaluates it and returns the typed value. It returns:
// - value: the evaluated value of the single expression
// - isSingle: whether the template is a single expression template
// - wasUndefined: whether the expression evaluated to a strictly undefined value
// - err: any parsing or evaluation error encountered
func TryEvaluateSingleExpressionTemplate(template string, context map[string]interface{}) (interface{}, bool, bool, error) {
	// Parse the template (reusing the same parser logic as TemplateString)
	nodes, found := defaultTemplateCache.Get(template)
	if !found {
		parser := NewParser(template)
		var err error
		nodes, err = parser.ParseAll()
		if err != nil {
			return nil, false, false, fmt.Errorf("template parsing error: %w", err)
		}
		defaultTemplateCache.Set(template, nodes)
	}

	// Filter out whitespace-only text and comments
	filtered := make([]*Node, 0, len(nodes))
	for _, n := range nodes {
		switch n.Type {
		case NodeComment:
			// ignore
			continue
		case NodeText:
			if strings.TrimSpace(n.Content) == "" {
				continue
			}
			// Non-whitespace text means it's not a single expression template
			filtered = append(filtered, n)
		default:
			filtered = append(filtered, n)
		}
	}

	if len(filtered) != 1 || filtered[0].Type != NodeExpression {
		return nil, false, false, nil
	}

	// Evaluate the single expression using the internal evaluator to obtain typed value
	val, wasStrictlyUndefined, err := evaluateFullExpressionInternal(filtered[0].Content, context)
	if err != nil {
		return nil, true, false, fmt.Errorf("error evaluating expression '{{ %s }}': %v", filtered[0].Content, err)
	}
	return val, true, wasStrictlyUndefined, nil
}

func processExpression(node *Node, context map[string]any, result *strings.Builder) error {
	// With the new LALR-based `evaluateFullExpressionInternal`, we can process all expressions uniformly.
	val, wasUndefined, err := evaluateFullExpressionInternal(node.Content, context)
	if err != nil {
		return fmt.Errorf("error evaluating expression '{{ %s }}': %v", node.Content, err)
	}

	if wasUndefined {
		// Jinja2 renders undefined variables as empty strings
		return nil
	}

	switch v := val.(type) {
	case string:
		result.WriteString(v)
	case nil:
		// nil values render as empty strings
		// Do nothing, no output
	case OmitType:
		// Omit values should not be rendered at all
		// Do nothing, no output
	default:
		// Use the generic Python-style formatter for all types
		result.WriteString(formatPythonStyle(v))
	}
	return nil
}

// processNodes recursively processes a slice of nodes, handling control flow like {% if %}.
func processNodes(nodes []*Node, context map[string]interface{}) (string, error) {
	return processNodesWithResolver(nodes, context, defaultFileResolver)
}

// processNodesWithResolver recursively processes nodes using a resolver for include
func processNodesWithResolver(nodes []*Node, context map[string]interface{}, resolver FileResolver) (string, error) {
	var result strings.Builder
	currentIndex := 0

	for currentIndex < len(nodes) {
		node := nodes[currentIndex]

		switch node.Type {
		case NodeText:
			result.WriteString(node.Content)
			currentIndex++
		case NodeExpression:
			if err := processExpression(node, context, &result); err != nil {
				return "", err
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
			case ControlInclude:
				includeExpr := strings.TrimSpace(node.Control.Expression)
				if includeExpr == "" {
					return "", fmt.Errorf("include tag requires a template path or expression")
				}
				val, err := EvaluateExpression(includeExpr, context)
				if err != nil {
					return "", fmt.Errorf("error evaluating include expression '%s': %v", includeExpr, err)
				}
				path, ok := val.(string)
				if !ok {
					return "", fmt.Errorf("include expression must evaluate to a string path, got %T", val)
				}
				contentBytes, err := resolver(path)
				if err != nil {
					return "", fmt.Errorf("failed to read included template '%s': %v", path, err)
				}
				// Parse and render the included content with the same resolver
				p := NewParser(string(contentBytes))
				includedNodes, perr := p.ParseAll()
				if perr != nil {
					return "", perr
				}
				var buf strings.Builder
				if err := renderNodesWithResolver(includedNodes, context, &buf, resolver); err != nil {
					return "", err
				}
				result.WriteString(buf.String())
				currentIndex++
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

// renderNodesWithResolver processes nodes with a resolver for file-based features
func renderNodesWithResolver(nodes []*Node, context map[string]interface{}, sb *strings.Builder, resolver FileResolver) error {
	result, err := processNodesWithResolver(nodes, context, resolver)
	if err != nil {
		return err
	}
	sb.WriteString(result)
	return nil
}

// EvaluateExpression evaluates a single expression string (without surrounding {{ }})
// against the provided context. It applies filters as specified.
// If the variable is undefined after evaluation (and not handled by a filter like default),
// an error is returned.
func EvaluateExpression(expression string, context map[string]interface{}) (interface{}, error) {
	trimmedExpression := strings.TrimSpace(expression)

	// `evaluateFullExpressionInternal` now uses the LALR parser, so we can just call it.
	val, wasStrictlyUndefined, err := evaluateFullExpressionInternal(trimmedExpression, context)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %v", expression, err)
	}

	if wasStrictlyUndefined {
		// For EvaluateExpression, strictly undefined (and not resolved by a filter like default)
		// should be an error, as per the project requirements.
		return nil, fmt.Errorf("variable in expression '%s' is undefined", expression)
	}

	return val, nil
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

// ParseVariablesFromExpression extracts all root Jinja variable names from a Jinja expression string.
// For example, for the expression `item.some_key`, it returns ["item"].
// For `item.some_bool and another_item`, it returns ["item", "another_item"].
func ParseVariablesFromExpression(expression string) ([]string, error) {
	variableSet := make(map[string]bool)
	err := extractVariablesFromExpression(expression, variableSet)
	if err != nil {
		return nil, err
	}
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
				case ControlInclude:
					// The include expression could be a literal string or an expression like variable
					if node.Control.Expression != "" {
						// Try to extract variables from the include expression as well
						_ = extractVariablesFromExpression(node.Control.Expression, variableSet)
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
		variableSet[node.Identifier] = true
	case NodeAttribute:
		if len(node.Children) > 0 {
			extractVariablesFromAST(node.Children[0], variableSet)
		}
	case NodeSubscript:
		if len(node.Children) > 0 {
			extractVariablesFromAST(node.Children[0], variableSet)
		}
		if len(node.Children) > 1 {
			extractVariablesFromAST(node.Children[1], variableSet)
		}
	case NodeFunctionCall:
		for _, child := range node.Children[1:] {
			extractVariablesFromAST(child, variableSet)
		}
	case NodeList, NodeTuple:
		for _, child := range node.Children {
			extractVariablesFromAST(child, variableSet)
		}
	case NodeDict:
		for _, child := range node.Children {
			extractVariablesFromAST(child, variableSet)
		}
	case NodeUnaryOp:
		for _, child := range node.Children {
			extractVariablesFromAST(child, variableSet)
		}
	case NodeBinaryOp:
		extractVariablesFromAST(node.Children[0], variableSet)

		// Only extract variables from the right hand side of the expression
		// on 'is' or 'is not', if it's not a test function like 'defined'
		if node.Operator == "is" || node.Operator == "is not" {
			if _, ok := GlobalTests[node.Children[1].Identifier]; !ok {
				extractVariablesFromAST(node.Children[1], variableSet)
			}
		} else {
			extractVariablesFromAST(node.Children[1], variableSet)
		}
	case NodeFilterChain:
		// Main expression is first child
		if len(node.Children) > 0 {
			extractVariablesFromAST(node.Children[0], variableSet)
		}
		// Filter arguments
		for _, arg := range node.FilterArgs {
			extractVariablesFromAST(arg, variableSet)
		}
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
		// Skip test functions like 'defined'
		if _, ok := GlobalTests[match]; ok {
			continue
		}
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

// formatPythonStyle formats any value in Python-style string representation
// e.g., [1, "abc", true] -> "[1, \"abc\", true]"
// e.g., {"a": 1, "b": true} -> "{\"a\": 1, \"b\": true}"
func formatPythonStyle(val interface{}) string {
	if val == nil {
		return "null"
	}

	switch v := val.(type) {
	case string:
		// Quote strings
		return fmt.Sprintf("%q", v)
	case bool:
		// Boolean values as lowercase
		return fmt.Sprintf("%t", v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		// Integer types
		return fmt.Sprintf("%v", v)
	case float32, float64:
		// Float types
		return fmt.Sprintf("%v", v)
	case []interface{}:
		// Lists
		if len(v) == 0 {
			return "[]"
		}
		var parts []string
		for _, item := range v {
			parts = append(parts, formatPythonStyle(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]interface{}:
		// Dictionaries
		if len(v) == 0 {
			return "{}"
		}
		var parts []string

		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts = make([]string, 0, len(v))
		for _, key := range keys {
			parts = append(parts, fmt.Sprintf("%q: %s", key, formatPythonStyle(v[key])))
		}

		return "{" + strings.Join(parts, ", ") + "}"

	default:
		// Use reflection for other types
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Slice, reflect.Array:
			// Handle other slice/array types
			if rv.Len() == 0 {
				return "[]"
			}
			var parts []string
			for i := 0; i < rv.Len(); i++ {
				parts = append(parts, formatPythonStyle(rv.Index(i).Interface()))
			}
			return "[" + strings.Join(parts, ", ") + "]"
		case reflect.Map:
			// Handle other map types
			if rv.Len() == 0 {
				return "{}"
			}
			var parts []string
			iter := rv.MapRange()
			for iter.Next() {
				key := formatPythonStyle(iter.Key().Interface())
				value := formatPythonStyle(iter.Value().Interface())
				parts = append(parts, fmt.Sprintf("%s: %s", key, value))
			}
			// Don't sort for consistent output - preserve order
			return "{" + strings.Join(parts, ", ") + "}"
		case reflect.Ptr:
			// Handle pointers
			if rv.IsNil() {
				return "null"
			}
			return formatPythonStyle(rv.Elem().Interface())
		default:
			// For other types, use their default string representation
			return fmt.Sprintf("%v", v)
		}
	}
}
