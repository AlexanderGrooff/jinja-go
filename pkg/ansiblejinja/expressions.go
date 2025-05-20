package ansiblejinja

/*
This file implements a LALR (Look-Ahead LR) parser for Jinja expressions.
The implementation follows a three-phase approach:

1. Lexical Analysis (Lexer):
   - Scans the input string and converts it to a sequence of tokens
   - Handles literals, operators, identifiers, and other lexical elements

2. Syntactic Analysis (Parser):
   - Parses the token stream into an Abstract Syntax Tree (AST)
   - Implements operator precedence and associativity
   - Handles complex nested expressions

3. Semantic Analysis (Evaluator):
   - Evaluates the AST against a provided context
   - Implements operator semantics and variable resolution
   - Returns the final computed value

This approach provides better performance, maintainability, and error handling
compared to recursive descent parsing, especially for complex expressions.
*/

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

// ExprNodeType represents the type of AST node
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

// Lexer breaks input string into tokens
type Lexer struct {
	input      string
	pos        int
	currentPos int
	tokens     []Token
}

// NewLexer creates a new lexer instance
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:      input,
		pos:        0,
		currentPos: 0,
		tokens:     []Token{},
	}
}

// Tokenize breaks the input string into tokens
func (l *Lexer) Tokenize() ([]Token, error) {
	l.tokens = []Token{}
	l.pos = 0
	input := strings.TrimSpace(l.input)

	for l.pos < len(input) {
		if isWhitespace(input[l.pos]) {
			l.pos++
			continue
		}

		// Single-character tokens
		switch input[l.pos] {
		case '(':
			l.addToken(TokenLeftParen, "(")
			continue
		case ')':
			l.addToken(TokenRightParen, ")")
			continue
		case '[':
			l.addToken(TokenLeftBracket, "[")
			continue
		case ']':
			l.addToken(TokenRightBracket, "]")
			continue
		case '{':
			l.addToken(TokenLeftBrace, "{")
			continue
		case '}':
			l.addToken(TokenRightBrace, "}")
			continue
		case ',':
			l.addToken(TokenComma, ",")
			continue
		case '.':
			l.addToken(TokenDot, ".")
			continue
		case ':':
			l.addToken(TokenColon, ":")
			continue
		case '|':
			l.addToken(TokenPipe, "|")
			continue
		}

		// String literals
		if input[l.pos] == '\'' || input[l.pos] == '"' {
			if err := l.tokenizeString(); err != nil {
				return nil, err
			}
			continue
		}

		// Numbers
		if isDigit(input[l.pos]) {
			l.tokenizeNumber()
			continue
		}

		// Check for multi-character operators
		if l.tryTokenizeOperator() {
			continue
		}

		// Keywords and identifiers
		if isAlpha(input[l.pos]) || input[l.pos] == '_' {
			l.tokenizeIdentifierOrKeyword()
			continue
		}

		// If we get here, we encountered an unexpected character
		return nil, fmt.Errorf("unexpected character '%c' at position %d", input[l.pos], l.pos)
	}

	// Add an EOF token
	l.tokens = append(l.tokens, Token{Type: TokenEOF, Value: "", Position: len(input)})
	return l.tokens, nil
}

// addToken adds a token to the token list and advances position
func (l *Lexer) addToken(tokenType TokenType, value string) {
	l.tokens = append(l.tokens, Token{Type: tokenType, Value: value, Position: l.pos})
	l.pos += len(value)
}

// tokenizeString handles string literals
func (l *Lexer) tokenizeString() error {
	quoteChar := l.input[l.pos]
	start := l.pos
	l.pos++ // Skip the opening quote

	for l.pos < len(l.input) && l.input[l.pos] != quoteChar {
		// Handle escape sequences
		if l.input[l.pos] == '\\' && l.pos+1 < len(l.input) {
			l.pos += 2 // Skip the backslash and the escaped character
		} else {
			l.pos++
		}
	}

	if l.pos >= len(l.input) {
		return fmt.Errorf("unterminated string literal at position %d", start)
	}

	// Include the closing quote
	l.pos++
	strLiteral := l.input[start:l.pos]
	l.tokens = append(l.tokens, Token{Type: TokenLiteral, Value: strLiteral, Position: start})
	return nil
}

// tokenizeNumber handles numeric literals
func (l *Lexer) tokenizeNumber() {
	start := l.pos
	hasDot := false

	for l.pos < len(l.input) && (isDigit(l.input[l.pos]) || (l.input[l.pos] == '.' && !hasDot)) {
		if l.input[l.pos] == '.' {
			hasDot = true
		}
		l.pos++
	}

	numStr := l.input[start:l.pos]
	l.tokens = append(l.tokens, Token{Type: TokenLiteral, Value: numStr, Position: start})
}

// tryTokenizeOperator attempts to tokenize an operator
func (l *Lexer) tryTokenizeOperator() bool {
	// Try special operators first
	if l.pos+6 <= len(l.input) && l.input[l.pos:l.pos+6] == "not in" {
		l.addToken(TokenOperator, "not in")
		return true
	}

	if l.pos+6 <= len(l.input) && l.input[l.pos:l.pos+6] == "is not" {
		l.addToken(TokenOperator, "is not")
		return true
	}

	// Try two-character operators
	if l.pos+2 <= len(l.input) {
		twoChars := l.input[l.pos : l.pos+2]
		if _, found := operators[twoChars]; found {
			l.addToken(TokenOperator, twoChars)
			return true
		}
	}

	// Try "is" operator
	if l.pos+2 <= len(l.input) && l.input[l.pos:l.pos+2] == "is" &&
		(l.pos+2 >= len(l.input) || !isAlpha(l.input[l.pos+2])) {
		l.addToken(TokenOperator, "is")
		return true
	}

	// Try single-character operators
	if _, found := operators[string(l.input[l.pos])]; found {
		l.addToken(TokenOperator, string(l.input[l.pos]))
		return true
	}

	return false
}

// tokenizeIdentifierOrKeyword handles identifiers and keywords
func (l *Lexer) tokenizeIdentifierOrKeyword() {
	start := l.pos
	for l.pos < len(l.input) && (isAlphaNumeric(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}

	word := l.input[start:l.pos]

	// Check if it's a keyword operator
	if _, found := operators[word]; found {
		l.tokens = append(l.tokens, Token{Type: TokenOperator, Value: word, Position: start})
	} else if word == "True" || word == "False" || word == "None" {
		// Handle boolean literals and None
		l.tokens = append(l.tokens, Token{Type: TokenLiteral, Value: word, Position: start})
	} else {
		// It's an identifier
		l.tokens = append(l.tokens, Token{Type: TokenIdentifier, Value: word, Position: start})
	}
}

// Helper functions for character classification
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

// Parser implements a LALR parser for Jinja expressions
type ExprParser struct {
	tokens   []Token
	pos      int
	symTable map[string]int
}

// NewParser creates a new Parser instance
func NewExprParser(tokens []Token) *ExprParser {
	return &ExprParser{
		tokens:   tokens,
		pos:      0,
		symTable: make(map[string]int),
	}
}

// Parse converts tokens to an AST
func (p *ExprParser) Parse() (*ExprNode, error) {
	return p.parseExpression(0)
}

// parseExpression parses an expression with given precedence
func (p *ExprParser) parseExpression(precedence int) (*ExprNode, error) {
	var left *ExprNode
	var err error

	// Parse the left-hand side of the expression
	if p.pos >= len(p.tokens) {
		return nil, fmt.Errorf("unexpected end of expression")
	}

	token := p.tokens[p.pos]
	p.pos++

	// Handle prefix operators and primary expressions
	switch token.Type {
	case TokenLiteral:
		left, err = p.parseLiteral(token)
	case TokenIdentifier:
		left, err = p.parseIdentifier(token)
	case TokenLeftParen:
		left, err = p.parseGrouping()
	case TokenLeftBracket:
		left, err = p.parseListLiteral()
	case TokenLeftBrace:
		left, err = p.parseDictLiteral()
	case TokenOperator:
		// Handle unary operators
		if token.Value == "not" || token.Value == "+" || token.Value == "-" {
			operand, err := p.parseExpression(operatorPrecedence[token.Value])
			if err != nil {
				return nil, err
			}
			left = &ExprNode{
				Type:     NodeUnaryOp,
				Operator: token.Value,
				Children: []*ExprNode{operand},
			}
		} else {
			return nil, fmt.Errorf("unexpected operator: %s", token.Value)
		}
	default:
		return nil, fmt.Errorf("unexpected token: %s", token.Value)
	}

	if err != nil {
		return nil, err
	}

	// Handle postfix operators and infix operators
	for p.pos < len(p.tokens) {
		if p.pos >= len(p.tokens) {
			break
		}

		token = p.tokens[p.pos]

		// Handle attribute access, subscript, or function call
		if token.Type == TokenDot {
			p.pos++
			left, err = p.parseAttributeAccess(left)
		} else if token.Type == TokenLeftBracket {
			p.pos++
			left, err = p.parseSubscriptAccess(left)
		} else if token.Type == TokenLeftParen {
			p.pos++
			left, err = p.parseFunctionCall(left)
		} else if token.Type == TokenOperator {
			// Process binary operators based on precedence
			opToken := token
			opPrecedence, ok := operatorPrecedence[opToken.Value]
			if !ok || opPrecedence < precedence {
				break
			}

			p.pos++ // Consume the operator

			// Parse the right-hand side with higher precedence
			right, err := p.parseExpression(opPrecedence + 1)
			if err != nil {
				return nil, err
			}

			left = &ExprNode{
				Type:     NodeBinaryOp,
				Operator: opToken.Value,
				Children: []*ExprNode{left, right},
			}
		} else {
			break
		}

		if err != nil {
			return nil, err
		}
	}

	return left, nil
}

// parseLiteral parses literal values
func (p *ExprParser) parseLiteral(token Token) (*ExprNode, error) {
	var value interface{}
	var err error

	// Check for quoted strings
	if (strings.HasPrefix(token.Value, "'") && strings.HasSuffix(token.Value, "'")) ||
		(strings.HasPrefix(token.Value, "\"") && strings.HasSuffix(token.Value, "\"")) {
		// Remove quotes and unescape
		value = unescapeStringLiteral(token.Value[1 : len(token.Value)-1])
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
}

// parseIdentifier parses an identifier
func (p *ExprParser) parseIdentifier(token Token) (*ExprNode, error) {
	return &ExprNode{
		Type:       NodeIdentifier,
		Identifier: token.Value,
	}, nil
}

// parseGrouping parses a parenthesized expression
func (p *ExprParser) parseGrouping() (*ExprNode, error) {
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenRightParen {
		return nil, fmt.Errorf("expected ')'")
	}
	p.pos++ // Consume the right parenthesis

	return expr, nil
}

// parseAttributeAccess parses an attribute access expression (obj.attr)
func (p *ExprParser) parseAttributeAccess(left *ExprNode) (*ExprNode, error) {
	if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenIdentifier {
		return nil, fmt.Errorf("expected identifier after '.'")
	}

	attrName := p.tokens[p.pos].Value
	p.pos++ // Consume the attribute name

	return &ExprNode{
		Type:       NodeAttribute,
		Identifier: attrName,
		Children:   []*ExprNode{left},
	}, nil
}

// parseSubscriptAccess parses a subscript access expression (obj[key])
func (p *ExprParser) parseSubscriptAccess(left *ExprNode) (*ExprNode, error) {
	// Parse the key expression
	keyExpr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenRightBracket {
		return nil, fmt.Errorf("expected ']'")
	}
	p.pos++ // Consume the right bracket

	return &ExprNode{
		Type:     NodeSubscript,
		Children: []*ExprNode{left, keyExpr},
	}, nil
}

// parseFunctionCall parses a function call expression (func(arg1, arg2))
func (p *ExprParser) parseFunctionCall(left *ExprNode) (*ExprNode, error) {
	var args []*ExprNode

	// Check for empty argument list
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenRightParen {
		p.pos++ // Consume the right parenthesis
	} else {
		// Parse arguments
		for {
			arg, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if p.pos >= len(p.tokens) {
				return nil, fmt.Errorf("unexpected end of expression, expected ')' or ','")
			}

			if p.tokens[p.pos].Type == TokenRightParen {
				p.pos++ // Consume the right parenthesis
				break
			}

			if p.tokens[p.pos].Type != TokenComma {
				return nil, fmt.Errorf("expected ',' or ')', found %s", p.tokens[p.pos].Value)
			}
			p.pos++ // Consume the comma
		}
	}

	return &ExprNode{
		Type:     NodeFunctionCall,
		Children: append([]*ExprNode{left}, args...),
	}, nil
}

// parseListLiteral parses a list literal expression [item1, item2, ...]
func (p *ExprParser) parseListLiteral() (*ExprNode, error) {
	var items []*ExprNode

	// Check for empty list
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenRightBracket {
		p.pos++ // Consume the right bracket
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

		if p.pos >= len(p.tokens) {
			return nil, fmt.Errorf("unexpected end of expression, expected ']' or ','")
		}

		if p.tokens[p.pos].Type == TokenRightBracket {
			p.pos++ // Consume the right bracket
			break
		}

		if p.tokens[p.pos].Type != TokenComma {
			return nil, fmt.Errorf("expected ',' or ']', found %s", p.tokens[p.pos].Value)
		}
		p.pos++ // Consume the comma
	}

	return &ExprNode{
		Type:     NodeList,
		Children: items,
	}, nil
}

// parseDictLiteral parses a dictionary literal expression {key: value, ...}
func (p *ExprParser) parseDictLiteral() (*ExprNode, error) {
	var keyValuePairs []*ExprNode

	// Check for empty dictionary
	if p.pos < len(p.tokens) && p.tokens[p.pos].Type == TokenRightBrace {
		p.pos++ // Consume the right brace
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

		if p.pos >= len(p.tokens) || p.tokens[p.pos].Type != TokenColon {
			return nil, fmt.Errorf("expected ':' after dictionary key")
		}
		p.pos++ // Consume the colon

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

		if p.pos >= len(p.tokens) {
			return nil, fmt.Errorf("unexpected end of expression, expected '}' or ','")
		}

		if p.tokens[p.pos].Type == TokenRightBrace {
			p.pos++ // Consume the right brace
			break
		}

		if p.tokens[p.pos].Type != TokenComma {
			return nil, fmt.Errorf("expected ',' or '}', found %s", p.tokens[p.pos].Value)
		}
		p.pos++ // Consume the comma
	}

	return &ExprNode{
		Type:     NodeDict,
		Children: keyValuePairs,
	}, nil
}

// Evaluator evaluates an AST with a given context
type Evaluator struct {
	context map[string]interface{}
}

// NewEvaluator creates a new evaluator with a context
func NewEvaluator(context map[string]interface{}) *Evaluator {
	return &Evaluator{context: context}
}

// Evaluate evaluates an AST node with context
func (e *Evaluator) Evaluate(node *ExprNode) (interface{}, error) {
	if node == nil {
		return nil, fmt.Errorf("cannot evaluate nil node")
	}

	switch node.Type {
	case NodeLiteral:
		return node.Value, nil

	case NodeIdentifier:
		value, exists := e.context[node.Identifier]
		if !exists {
			return nil, fmt.Errorf("variable '%s' is undefined", node.Identifier)
		}
		return value, nil

	case NodeUnaryOp:
		if len(node.Children) != 1 {
			return nil, fmt.Errorf("unary operator '%s' requires exactly one operand", node.Operator)
		}

		operand, err := e.Evaluate(node.Children[0])
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
		left, err := e.Evaluate(node.Children[0])
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
		right, err := e.Evaluate(node.Children[1])
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

		obj, err := e.Evaluate(node.Children[0])
		if err != nil {
			return nil, err
		}

		return getAttributeValue(obj, node.Identifier)

	case NodeSubscript:
		if len(node.Children) != 2 {
			return nil, fmt.Errorf("subscript access requires an object and key")
		}

		obj, err := e.Evaluate(node.Children[0])
		if err != nil {
			return nil, err
		}

		key, err := e.Evaluate(node.Children[1])
		if err != nil {
			return nil, err
		}

		return getSubscriptValue(obj, key)

	case NodeFunctionCall:
		if len(node.Children) < 1 {
			return nil, fmt.Errorf("function call requires a callable")
		}

		callable, err := e.Evaluate(node.Children[0])
		if err != nil {
			return nil, err
		}

		// Evaluate arguments
		var args []interface{}
		for i := 1; i < len(node.Children); i++ {
			arg, err := e.Evaluate(node.Children[i])
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
			item, err := e.Evaluate(child)
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

			pair, err := e.Evaluate(child)
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
	// Step 1: Tokenize the input
	lexer := NewLexer(expr)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("lexical error: %v", err)
	}

	// Step 2: Parse tokens into AST
	parser := NewExprParser(tokens)
	ast, err := parser.Parse()
	if err != nil {
		return nil, fmt.Errorf("syntax error: %v", err)
	}

	// Step 3: Evaluate the AST
	evaluator := NewEvaluator(context)
	result, err := evaluator.Evaluate(ast)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %v", err)
	}

	return result, nil
}

// unescapeString handles basic unescaping for string literals
func unescapeStringLiteral(s string) string {
	// Handle escaped characters
	s = strings.ReplaceAll(s, "\\'", "'")   // Escaped single quote
	s = strings.ReplaceAll(s, "\\\"", "\"") // Escaped double quote
	s = strings.ReplaceAll(s, "\\\\", "\\") // Escaped backslash
	s = strings.ReplaceAll(s, "\\n", "\n")  // Escaped newline
	s = strings.ReplaceAll(s, "\\t", "\t")  // Escaped tab
	s = strings.ReplaceAll(s, "\\r", "\r")  // Escaped carriage return
	return s
}

// evaluateCompoundExpression evaluates an expression with potential compound operations
func evaluateCompoundExpression(expr string, context map[string]interface{}) (interface{}, error) {
	// With our improved LALR parser, we can handle complex expressions directly
	return ParseAndEvaluate(expr, context)
}

// containsSubscript checks if the expression contains subscript operations
func containsSubscript(expr string) bool {
	return strings.Contains(expr, "[") && strings.Contains(expr, "]")
}
