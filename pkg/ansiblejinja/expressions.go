package ansiblejinja

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// TokenType represents different types of tokens in a Jinja expression
type TokenType int

const (
	TokenLiteral TokenType = iota
	TokenIdentifier
	TokenOperator
	TokenLeftParen
	TokenRightParen
	TokenLeftBracket
	TokenRightBracket
	TokenLeftBrace
	TokenRightBrace
	TokenComma
	TokenDot
	TokenColon
	TokenPipe
	TokenEOF
)

// Token represents a lexical token in a Jinja expression
type Token struct {
	Type     TokenType
	Value    string
	Position int
}

// List of operators ordered by increasing precedence
var operators = map[string]struct{}{
	// Logical operators
	"or":  {},
	"and": {},
	"not": {},

	// Comparison operators
	"==": {}, "!=": {}, ">=": {}, "<=": {}, ">": {}, "<": {},
	"in": {}, "not in": {}, "is": {}, "is not": {},

	// Mathematical operators
	"+": {}, "-": {},
	"*": {}, "/": {}, "//": {}, "%": {},
	"**": {},
}

// Operator precedence - higher number means higher precedence
var operatorPrecedence = map[string]int{
	"or":  10,
	"and": 20,
	"not": 30,
	"==":  40, "!=": 40, ">=": 40, "<=": 40, ">": 40, "<": 40,
	"in": 40, "not in": 40, "is": 40, "is not": 40,
	"+": 50, "-": 50,
	"*": 60, "/": 60, "//": 60, "%": 60,
	"**": 70,
}

// NodeType represents the type of AST node
type ExprNodeType int

const (
	NodeLiteral ExprNodeType = iota
	NodeIdentifier
	NodeUnaryOp
	NodeBinaryOp
	NodeAttribute
	NodeSubscript
	NodeFunctionCall
	NodeList
	NodeDict
	NodeTuple
)

// ExprNode represents a node in the expression AST
type ExprNode struct {
	Type       ExprNodeType
	Value      interface{}
	Children   []*ExprNode
	Operator   string
	Identifier string
}

// ExpressionParser parses and evaluates Jinja expressions
type ExpressionParser struct {
	input     string
	tokens    []Token
	position  int
	currentOp string
}

// NewExpressionParser creates a new expression parser
func NewExpressionParser(input string) *ExpressionParser {
	return &ExpressionParser{
		input:    input,
		tokens:   []Token{},
		position: 0,
	}
}

// tokenize breaks the input string into tokens
func (p *ExpressionParser) tokenize() error {
	p.tokens = []Token{}
	pos := 0
	input := strings.TrimSpace(p.input)

	for pos < len(input) {
		if input[pos] == ' ' || input[pos] == '\t' || input[pos] == '\n' || input[pos] == '\r' {
			// Skip whitespace
			pos++
			continue
		}

		// Single-character tokens
		switch input[pos] {
		case '(':
			p.tokens = append(p.tokens, Token{Type: TokenLeftParen, Value: "(", Position: pos})
			pos++
			continue
		case ')':
			p.tokens = append(p.tokens, Token{Type: TokenRightParen, Value: ")", Position: pos})
			pos++
			continue
		case '[':
			p.tokens = append(p.tokens, Token{Type: TokenLeftBracket, Value: "[", Position: pos})
			pos++
			continue
		case ']':
			p.tokens = append(p.tokens, Token{Type: TokenRightBracket, Value: "]", Position: pos})
			pos++
			continue
		case '{':
			p.tokens = append(p.tokens, Token{Type: TokenLeftBrace, Value: "{", Position: pos})
			pos++
			continue
		case '}':
			p.tokens = append(p.tokens, Token{Type: TokenRightBrace, Value: "}", Position: pos})
			pos++
			continue
		case ',':
			p.tokens = append(p.tokens, Token{Type: TokenComma, Value: ",", Position: pos})
			pos++
			continue
		case '.':
			p.tokens = append(p.tokens, Token{Type: TokenDot, Value: ".", Position: pos})
			pos++
			continue
		case ':':
			p.tokens = append(p.tokens, Token{Type: TokenColon, Value: ":", Position: pos})
			pos++
			continue
		case '|':
			p.tokens = append(p.tokens, Token{Type: TokenPipe, Value: "|", Position: pos})
			pos++
			continue
		}

		// String literals
		if input[pos] == '\'' || input[pos] == '"' {
			quoteChar := input[pos]
			start := pos
			pos++ // Skip the opening quote

			for pos < len(input) && input[pos] != quoteChar {
				// Handle escape sequences
				if input[pos] == '\\' && pos+1 < len(input) {
					pos += 2 // Skip the backslash and the escaped character
				} else {
					pos++
				}
			}

			if pos >= len(input) {
				return fmt.Errorf("unterminated string literal at position %d", start)
			}

			// Include the closing quote
			pos++
			strLiteral := input[start:pos]
			p.tokens = append(p.tokens, Token{Type: TokenLiteral, Value: strLiteral, Position: start})
			continue
		}

		// Numbers
		if isDigit(input[pos]) {
			start := pos
			hasDot := false

			for pos < len(input) && (isDigit(input[pos]) || (input[pos] == '.' && !hasDot)) {
				if input[pos] == '.' {
					hasDot = true
				}
				pos++
			}

			numStr := input[start:pos]
			p.tokens = append(p.tokens, Token{Type: TokenLiteral, Value: numStr, Position: start})
			continue
		}

		// Multi-character operators: ==, !=, >=, <=, **, //, not in, is not
		if pos+1 < len(input) {
			twoChars := input[pos : pos+2]
			if _, found := operators[twoChars]; found {
				p.tokens = append(p.tokens, Token{Type: TokenOperator, Value: twoChars, Position: pos})
				pos += 2
				continue
			}
		}

		// Check for longer operators like "not in" and "is not"
		if pos+6 < len(input) && input[pos:pos+6] == "not in" &&
			(pos+6 >= len(input) || !isAlpha(input[pos+6])) {
			p.tokens = append(p.tokens, Token{Type: TokenOperator, Value: "not in", Position: pos})
			pos += 6
			continue
		}

		if pos+6 < len(input) && input[pos:pos+6] == "is not" &&
			(pos+6 >= len(input) || !isAlpha(input[pos+6])) {
			p.tokens = append(p.tokens, Token{Type: TokenOperator, Value: "is not", Position: pos})
			pos += 6
			continue
		}

		// Check for "is" operator
		if pos+2 < len(input) && input[pos:pos+2] == "is" &&
			(pos+2 >= len(input) || !isAlpha(input[pos+2])) {
			p.tokens = append(p.tokens, Token{Type: TokenOperator, Value: "is", Position: pos})
			pos += 2
			continue
		}

		// Single-character operators: +, -, *, /, %
		if _, found := operators[string(input[pos])]; found {
			p.tokens = append(p.tokens, Token{Type: TokenOperator, Value: string(input[pos]), Position: pos})
			pos++
			continue
		}

		// Keywords and identifiers
		if isAlpha(input[pos]) || input[pos] == '_' {
			start := pos
			for pos < len(input) && (isAlphaNumeric(input[pos]) || input[pos] == '_') {
				pos++
			}

			word := input[start:pos]

			// Check if it's a keyword operator
			if _, found := operators[word]; found {
				p.tokens = append(p.tokens, Token{Type: TokenOperator, Value: word, Position: start})
			} else if word == "True" || word == "False" || word == "None" {
				// Handle boolean literals and None
				p.tokens = append(p.tokens, Token{Type: TokenLiteral, Value: word, Position: start})
			} else {
				// It's an identifier
				p.tokens = append(p.tokens, Token{Type: TokenIdentifier, Value: word, Position: start})
			}
			continue
		}

		// If we get here, we encountered an unexpected character
		return fmt.Errorf("unexpected character '%c' at position %d", input[pos], pos)
	}

	// Add an EOF token
	p.tokens = append(p.tokens, Token{Type: TokenEOF, Value: "", Position: len(input)})
	return nil
}

// Helper functions for character classification
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

// parse converts tokens to an AST
func (p *ExpressionParser) parse() (*ExprNode, error) {
	p.position = 0
	return p.parseExpression(0)
}

// parseExpression parses an expression with given precedence
func (p *ExpressionParser) parseExpression(precedence int) (*ExprNode, error) {
	// Parse the left-hand side of the expression
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}

	// Keep going while the current operator has a higher precedence
	for p.position < len(p.tokens)-1 && p.tokens[p.position].Type == TokenOperator {
		opToken := p.tokens[p.position]

		// Special case for "is not" which is parsed as two separate tokens "is" and "not"
		if opToken.Value == "is" && p.position+1 < len(p.tokens) &&
			p.tokens[p.position+1].Type == TokenOperator && p.tokens[p.position+1].Value == "not" {
			opToken.Value = "is not"
			p.position++ // Skip the next token ("not")
		}

		opPrecedence, ok := operatorPrecedence[opToken.Value]
		if !ok || opPrecedence < precedence {
			break
		}

		// Consume the operator
		p.position++

		// Special case for unary operators like "not"
		if opToken.Value == "not" {
			// Create a unary operation node
			node := &ExprNode{
				Type:     NodeUnaryOp,
				Operator: opToken.Value,
				Children: []*ExprNode{left},
			}
			left = node
			continue
		}

		// Parse the right-hand side of the expression with higher precedence
		right, err := p.parseExpression(opPrecedence + 1)
		if err != nil {
			return nil, err
		}

		// Create a binary operation node
		left = &ExprNode{
			Type:     NodeBinaryOp,
			Operator: opToken.Value,
			Children: []*ExprNode{left, right},
		}
	}

	return left, nil
}

// parsePrimary parses primary expressions (literals, identifiers, parenthesized expressions)
func (p *ExpressionParser) parsePrimary() (*ExprNode, error) {
	if p.position >= len(p.tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	token := p.tokens[p.position]
	p.position++

	switch token.Type {
	case TokenLiteral:
		// Parse literal values (strings, numbers, boolean, None)
		var value interface{}
		var err error

		// Check for quoted strings
		if (strings.HasPrefix(token.Value, "'") && strings.HasSuffix(token.Value, "'")) ||
			(strings.HasPrefix(token.Value, "\"") && strings.HasSuffix(token.Value, "\"")) {
			// Remove quotes and unescape
			value = unescapeString(token.Value[1 : len(token.Value)-1])
		} else if token.Value == "True" {
			value = true
		} else if token.Value == "False" {
			value = false
		} else if token.Value == "None" {
			value = nil
		} else {
			// Try to parse as number
			if strings.Contains(token.Value, ".") {
				value, err = strconv.ParseFloat(token.Value, 64)
			} else {
				value, err = strconv.Atoi(token.Value)
			}

			if err != nil {
				return nil, fmt.Errorf("invalid literal: %s", token.Value)
			}
		}

		return &ExprNode{
			Type:  NodeLiteral,
			Value: value,
		}, nil

	case TokenIdentifier:
		identifier := token.Value
		node := &ExprNode{
			Type:       NodeIdentifier,
			Identifier: identifier,
		}

		// Check for attribute access (dot notation)
		if p.position < len(p.tokens) && p.tokens[p.position].Type == TokenDot {
			return p.parseAttributeAccess(node)
		}

		// Check for subscript access (dictionary/list lookup)
		if p.position < len(p.tokens) && p.tokens[p.position].Type == TokenLeftBracket {
			return p.parseSubscriptAccess(node)
		}

		// Check for function call
		if p.position < len(p.tokens) && p.tokens[p.position].Type == TokenLeftParen {
			return p.parseFunctionCall(node)
		}

		return node, nil

	case TokenLeftParen:
		// Parenthesized expression
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		if p.position >= len(p.tokens) || p.tokens[p.position].Type != TokenRightParen {
			return nil, fmt.Errorf("expected ')', found %s", p.tokens[p.position].Value)
		}
		p.position++ // Consume the right parenthesis

		// Check for attribute access, subscript, or function call after parenthesized expr
		if p.position < len(p.tokens) {
			if p.tokens[p.position].Type == TokenDot {
				return p.parseAttributeAccess(expr)
			} else if p.tokens[p.position].Type == TokenLeftBracket {
				return p.parseSubscriptAccess(expr)
			} else if p.tokens[p.position].Type == TokenLeftParen {
				return p.parseFunctionCall(expr)
			}
		}

		return expr, nil

	case TokenLeftBracket:
		// List literal [item1, item2, ...]
		return p.parseListLiteral()

	case TokenLeftBrace:
		// Dictionary literal {key: value, ...}
		return p.parseDictLiteral()

	case TokenOperator:
		// Handle unary operators (like -1, not x)
		if token.Value == "not" || token.Value == "+" || token.Value == "-" {
			expr, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			return &ExprNode{
				Type:     NodeUnaryOp,
				Operator: token.Value,
				Children: []*ExprNode{expr},
			}, nil
		}
		return nil, fmt.Errorf("unexpected operator: %s", token.Value)

	default:
		return nil, fmt.Errorf("unexpected token: %s", token.Value)
	}
}

// parseAttributeAccess parses an attribute access expression (obj.attr)
func (p *ExpressionParser) parseAttributeAccess(left *ExprNode) (*ExprNode, error) {
	p.position++ // Consume the dot

	if p.position >= len(p.tokens) || p.tokens[p.position].Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after '.'")
	}

	attrName := p.tokens[p.position].Value
	p.position++ // Consume the attribute name

	node := &ExprNode{
		Type:       NodeAttribute,
		Identifier: attrName,
		Children:   []*ExprNode{left},
	}

	// Check for further attribute access, subscript, or function call
	if p.position < len(p.tokens) {
		if p.tokens[p.position].Type == TokenDot {
			return p.parseAttributeAccess(node)
		} else if p.tokens[p.position].Type == TokenLeftBracket {
			return p.parseSubscriptAccess(node)
		} else if p.tokens[p.position].Type == TokenLeftParen {
			return p.parseFunctionCall(node)
		}
	}

	return node, nil
}

// parseSubscriptAccess parses a subscript access expression (obj[key])
func (p *ExpressionParser) parseSubscriptAccess(left *ExprNode) (*ExprNode, error) {
	p.position++ // Consume the left bracket

	// Parse the key expression
	keyExpr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.position >= len(p.tokens) || p.tokens[p.position].Type != TokenRightBracket {
		return nil, fmt.Errorf("expected ']'")
	}
	p.position++ // Consume the right bracket

	node := &ExprNode{
		Type:     NodeSubscript,
		Children: []*ExprNode{left, keyExpr},
	}

	// Check for further attribute access, subscript, or function call
	if p.position < len(p.tokens) {
		if p.tokens[p.position].Type == TokenDot {
			return p.parseAttributeAccess(node)
		} else if p.tokens[p.position].Type == TokenLeftBracket {
			return p.parseSubscriptAccess(node)
		} else if p.tokens[p.position].Type == TokenLeftParen {
			return p.parseFunctionCall(node)
		}
	}

	return node, nil
}

// parseFunctionCall parses a function call expression (func(arg1, arg2))
func (p *ExpressionParser) parseFunctionCall(left *ExprNode) (*ExprNode, error) {
	p.position++ // Consume the left parenthesis

	var args []*ExprNode

	// Check for empty argument list
	if p.position < len(p.tokens) && p.tokens[p.position].Type == TokenRightParen {
		p.position++ // Consume the right parenthesis
	} else {
		// Parse arguments
		for {
			arg, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.position >= len(p.tokens) {
				return nil, fmt.Errorf("unexpected end of expression, expected ')' or ','")
			}

			if p.tokens[p.position].Type == TokenRightParen {
				p.position++ // Consume the right parenthesis
				break
			}

			if p.tokens[p.position].Type != TokenComma {
				return nil, fmt.Errorf("expected ',' or ')', found %s", p.tokens[p.position].Value)
			}
			p.position++ // Consume the comma
		}
	}

	node := &ExprNode{
		Type:     NodeFunctionCall,
		Children: append([]*ExprNode{left}, args...),
	}

	// Check for further attribute access, subscript, or function call
	if p.position < len(p.tokens) {
		if p.tokens[p.position].Type == TokenDot {
			return p.parseAttributeAccess(node)
		} else if p.tokens[p.position].Type == TokenLeftBracket {
			return p.parseSubscriptAccess(node)
		} else if p.tokens[p.position].Type == TokenLeftParen {
			return p.parseFunctionCall(node)
		}
	}

	return node, nil
}

// parseListLiteral parses a list literal expression [item1, item2, ...]
func (p *ExpressionParser) parseListLiteral() (*ExprNode, error) {
	var items []*ExprNode

	// Check for empty list
	if p.position < len(p.tokens) && p.tokens[p.position].Type == TokenRightBracket {
		p.position++ // Consume the right bracket
		return &ExprNode{
			Type:     NodeList,
			Children: items,
		}, nil
	}

	// Parse list items
	for {
		item, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}
		items = append(items, item)

		if p.position >= len(p.tokens) {
			return nil, fmt.Errorf("unexpected end of expression, expected ']' or ','")
		}

		if p.tokens[p.position].Type == TokenRightBracket {
			p.position++ // Consume the right bracket
			break
		}

		if p.tokens[p.position].Type != TokenComma {
			return nil, fmt.Errorf("expected ',' or ']', found %s", p.tokens[p.position].Value)
		}
		p.position++ // Consume the comma
	}

	return &ExprNode{
		Type:     NodeList,
		Children: items,
	}, nil
}

// parseDictLiteral parses a dictionary literal expression {key: value, ...}
func (p *ExpressionParser) parseDictLiteral() (*ExprNode, error) {
	var keyValuePairs []*ExprNode

	// Check for empty dictionary
	if p.position < len(p.tokens) && p.tokens[p.position].Type == TokenRightBrace {
		p.position++ // Consume the right brace
		return &ExprNode{
			Type:     NodeDict,
			Children: keyValuePairs,
		}, nil
	}

	// Parse dictionary entries
	for {
		// Parse key
		key, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		if p.position >= len(p.tokens) || p.tokens[p.position].Type != TokenColon {
			return nil, fmt.Errorf("expected ':' after dictionary key")
		}
		p.position++ // Consume the colon

		// Parse value
		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		// Add key-value pair
		pair := &ExprNode{
			Type:     NodeBinaryOp,
			Operator: ":",
			Children: []*ExprNode{key, value},
		}
		keyValuePairs = append(keyValuePairs, pair)

		if p.position >= len(p.tokens) {
			return nil, fmt.Errorf("unexpected end of expression, expected '}' or ','")
		}

		if p.tokens[p.position].Type == TokenRightBrace {
			p.position++ // Consume the right brace
			break
		}

		if p.tokens[p.position].Type != TokenComma {
			return nil, fmt.Errorf("expected ',' or '}', found %s", p.tokens[p.position].Value)
		}
		p.position++ // Consume the comma
	}

	return &ExprNode{
		Type:     NodeDict,
		Children: keyValuePairs,
	}, nil
}

// evaluate computes the value of an AST node with context
func (p *ExpressionParser) evaluate(node *ExprNode, context map[string]interface{}) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("cannot evaluate nil node")
	}

	switch node.Type {
	case NodeLiteral:
		return node.Value, nil

	case NodeIdentifier:
		value, exists := context[node.Identifier]
		if !exists {
			return nil, fmt.Errorf("variable '%s' is undefined", node.Identifier)
		}
		return value, nil

	case NodeUnaryOp:
		if len(node.Children) != 1 {
			return nil, fmt.Errorf("unary operator '%s' requires exactly one operand", node.Operator)
		}

		operand, err := p.evaluate(node.Children[0], context)
		if err != nil {
			return nil, err
		}

		switch node.Operator {
		case "not":
			return !isTruthy(operand), nil
		case "+":
			// Unary plus - most types remain unchanged
			return operand, nil
		case "-":
			// Unary minus - negate numeric values
			return negateValue(operand)
		default:
			return nil, fmt.Errorf("unknown unary operator: %s", node.Operator)
		}

	case NodeBinaryOp:
		if len(node.Children) != 2 {
			return nil, fmt.Errorf("binary operator '%s' requires exactly two operands", node.Operator)
		}

		// Get the left operand
		left, err := p.evaluate(node.Children[0], context)
		if err != nil {
			return nil, err
		}

		// Short-circuit evaluation for 'and' and 'or'
		if node.Operator == "and" {
			if !isTruthy(left) {
				return left, nil
			}
		} else if node.Operator == "or" {
			if isTruthy(left) {
				return left, nil
			}
		}

		// Get the right operand
		right, err := p.evaluate(node.Children[1], context)
		if err != nil {
			return nil, err
		}

		// Handle the specific binary operator
		switch node.Operator {
		// Logical operators
		case "and":
			return right, nil // Left already evaluated and was truthy
		case "or":
			return right, nil // Left already evaluated and was falsy

		// Equality operators
		case "==":
			return equals(left, right)
		case "!=":
			eq, err := equals(left, right)
			if err != nil {
				return nil, err
			}
			return !eq.(bool), nil

		// Comparison operators
		case "<":
			return compare(left, right, func(a, b float64) bool { return a < b })
		case "<=":
			return compare(left, right, func(a, b float64) bool { return a <= b })
		case ">":
			return compare(left, right, func(a, b float64) bool { return a > b })
		case ">=":
			return compare(left, right, func(a, b float64) bool { return a >= b })

		// Membership operators
		case "in":
			return checkMembership(right, left)
		case "not in":
			result, err := checkMembership(right, left)
			if err != nil {
				return nil, err
			}
			return !result.(bool), nil

		// Identity operators
		case "is":
			// Use proper deep equality check
			result := reflect.DeepEqual(left, right)
			return result, nil
		case "is not":
			// Use proper deep equality check and negate the result
			result := !reflect.DeepEqual(left, right)
			return result, nil

		// Mathematical operators
		case "+":
			return add(left, right)
		case "-":
			return subtract(left, right)
		case "*":
			return multiply(left, right)
		case "/":
			return divide(left, right)
		case "//":
			return floorDivide(left, right)
		case "%":
			return modulo(left, right)
		case "**":
			return power(left, right)

		// Dictionary key-value separator (used in dict literals)
		case ":":
			// This is a special case used in dictionary construction
			// Just return a tuple of the key and value
			return []interface{}{left, right}, nil

		default:
			return nil, fmt.Errorf("unknown binary operator: %s", node.Operator)
		}

	case NodeAttribute:
		if len(node.Children) != 1 {
			return nil, fmt.Errorf("attribute access requires an object")
		}

		obj, err := p.evaluate(node.Children[0], context)
		if err != nil {
			return nil, err
		}

		return getAttributeValue(obj, node.Identifier)

	case NodeSubscript:
		if len(node.Children) != 2 {
			return nil, fmt.Errorf("subscript access requires an object and key")
		}

		obj, err := p.evaluate(node.Children[0], context)
		if err != nil {
			return nil, err
		}

		key, err := p.evaluate(node.Children[1], context)
		if err != nil {
			return nil, err
		}

		return getSubscriptValue(obj, key)

	case NodeFunctionCall:
		if len(node.Children) < 1 {
			return nil, fmt.Errorf("function call requires a callable")
		}

		callable, err := p.evaluate(node.Children[0], context)
		if err != nil {
			return nil, err
		}

		// Evaluate arguments
		var args []interface{}
		for i := 1; i < len(node.Children); i++ {
			arg, err := p.evaluate(node.Children[i], context)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)
		}

		return callFunction(callable, args)

	case NodeList:
		// Evaluate each item in the list
		var items []interface{}
		for _, child := range node.Children {
			item, err := p.evaluate(child, context)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil

	case NodeDict:
		// Create a map from key-value pairs
		dict := make(map[string]interface{})
		for _, child := range node.Children {
			if child.Type != NodeBinaryOp || child.Operator != ":" {
				return nil, fmt.Errorf("invalid dictionary entry: expected key-value pair")
			}

			pair, err := p.evaluate(child, context)
			if err != nil {
				return nil, err
			}

			keyValue, ok := pair.([]interface{})
			if !ok || len(keyValue) != 2 {
				return nil, fmt.Errorf("invalid dictionary entry format")
			}

			// In Jinja/Python, dictionary keys are usually strings, but can be any hashable type
			// For simplicity, we'll convert all keys to strings here
			key := fmt.Sprintf("%v", keyValue[0])
			dict[key] = keyValue[1]
		}
		return dict, nil

	default:
		return nil, fmt.Errorf("unknown node type: %v", node.Type)
	}
}

// ParseAndEvaluate parses and evaluates an expression string with context
func ParseAndEvaluate(expr string, context map[string]interface{}) (interface{}, error) {
	parser := NewExpressionParser(expr)

	if err := parser.tokenize(); err != nil {
		return nil, fmt.Errorf("lexical error: %v", err)
	}

	ast, err := parser.parse()
	if err != nil {
		return nil, fmt.Errorf("syntax error: %v", err)
	}

	result, err := parser.evaluate(ast, context)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %v", err)
	}

	return result, nil
}
