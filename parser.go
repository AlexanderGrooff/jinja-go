package jinja

import (
	"fmt"
	"strings"
)

var evaluateExpressionFunc func(expression string, context map[string]interface{}) (interface{}, error)

// Parser holds the state of the parsing process.
// It is used to iteratively parse a template string into a sequence of nodes.
type Parser struct {
	input string // The full template string being parsed.
	pos   int    // Current position in the input string.
	// Future additions could include: line/col numbers, error accumulation, etc.
}

// NewParser creates a new Parser instance for the given input string.
func NewParser(input string) *Parser {
	return &Parser{input: input, pos: 0}
}

// ControlTagType defines the specific type of a control tag.
type ControlTagType string

// Enumerates the different types of control tags.
const (
	ControlIf      ControlTagType = "if"
	ControlEndIf   ControlTagType = "endif"
	ControlFor     ControlTagType = "for"    // Placeholder for future 'for' loop implementation
	ControlEndFor  ControlTagType = "endfor" // Placeholder for future 'endfor' implementation
	ControlElse    ControlTagType = "else"   // Placeholder for future 'else' implementation
	ControlElseIf  ControlTagType = "elif"   // Placeholder for future 'elif' (else if) implementation
	ControlUnknown ControlTagType = "unknown"
)

// ControlTagInfo holds detailed information about a parsed control tag.
type ControlTagInfo struct {
	Type       ControlTagType
	Expression string // For 'if', 'elif', 'for': the condition or loop expression.
	// Future fields might include loop variables for 'for' tags.
}

// Node represents a parsed element in the template, such as literal text or an expression.
type Node struct {
	Type    NodeType        // The kind of node (e.g., text, expression).
	Content string          // The raw content of the node.
	Control *ControlTagInfo // Populated if Type is NodeControlTag, provides details about the control tag.
	// Future additions: original start/end positions, evaluated value for expressions, etc.
}

// NodeType defines the category of a parsed Node.
type NodeType int

// Enumerates the different types of nodes that can be encountered during parsing.
const (
	NodeText       NodeType = iota // Represents a segment of literal text.
	NodeExpression                 // Represents a Jinja expression, e.g., {{ variable }}.
	NodeComment                    // Represents a comment, e.g., {# comment #}.
	NodeControlTag                 // Represents a control structure tag, e.g., {% if ... %}.
)

// evaluateFullExpressionInternal is the core logic for evaluating an expression string,
// potentially including filters.
// It returns the final value, a boolean indicating if the base variable was strictly undefined
// (and not resolved by a filter like default), and an error if parsing/evaluation failed.
func evaluateFullExpressionInternal(fullExprStr string, context map[string]interface{}) (value interface{}, wasStrictlyUndefined bool, err error) {
	trimmedExpr := strings.TrimSpace(fullExprStr)
	if trimmedExpr == "" {
		// Handles `{{}}` or `{{   }}`. Jinja2 renders this as an empty string.
		// This is treated as an undefined variable named "".
		val, exists := context[""]
		if !exists {
			return nil, true, nil
		}
		return val, false, nil
	}

	// Use the LALR parser for all expression evaluation.
	val, err := ParseAndEvaluate(trimmedExpr, context)
	if err != nil {
		// A parsing or evaluation error occurred.
		return nil, false, err
	}

	// Check if the result is an unhandled undefined value.
	if _, ok := val.(UndefinedType); ok {
		return nil, true, nil
	}

	return val, false, nil
}

// parseCommentTag is called when "{#" is found.
// It extracts the content between "{#" and "#}".
func (p *Parser) parseCommentTag() *Node {
	originalPos := p.pos // For potential backtrack if parsing fails

	// Ensure we are actually at the start of a comment tag
	if !(p.pos+2 <= len(p.input) && p.input[p.pos:p.pos+2] == "{#") {
		return nil // Not a comment tag, or called incorrectly.
	}

	p.pos += 2                   // Consume "{#"
	commentContentStart := p.pos // The actual content starts after "{#"

	// Find the closing "#}"
	// Unlike expressions, comments typically don't have complex nesting rules
	// that require level counting or string literal skipping for the basic parsing of the comment block itself.
	// The content of the comment can be anything, including "{{" or other "{#".
	endMarkerIndex := strings.Index(p.input[p.pos:], "#}")
	if endMarkerIndex == -1 {
		// Error: unclosed comment tag.
		p.pos = originalPos // Backtrack
		return nil          // Indicate failure.
	}

	// If we reach here, a matching "#}" was found.
	// The content is from commentContentStart to p.pos + endMarkerIndex.
	content := p.input[commentContentStart : p.pos+endMarkerIndex]
	p.pos += endMarkerIndex + 2 // Advance parser position past "#}"

	return &Node{
		Type:    NodeComment,
		Content: content, // The content of the comment itself
	}
}

// parseControlTagDetail parses the trimmed content of a control tag (e.g., "if condition")
// and returns structured information about it.
func parseControlTagDetail(trimmedContent string) (*ControlTagInfo, error) {
	parts := strings.Fields(trimmedContent) // Splits by whitespace
	if len(parts) == 0 {
		// This case should ideally be prevented by prior checks ensuring content is not just whitespace,
		// or if it is, it implies an empty tag like {%%} which might be an error or specific syntax.
		return nil, fmt.Errorf("empty control tag content")
	}

	tagTypeStr := strings.ToLower(parts[0])
	info := &ControlTagInfo{}

	switch tagTypeStr {
	case "if":
		info.Type = ControlIf
		if len(parts) < 2 {
			return nil, fmt.Errorf("if tag requires a condition, e.g., {%% if user.isAdmin %%}")
		}
		// The rest of the parts form the expression.
		info.Expression = strings.Join(parts[1:], " ")
	case "endif":
		info.Type = ControlEndIf
		if len(parts) > 1 {
			// Jinja's {% endif %} typically does not take arguments.
			return nil, fmt.Errorf("endif tag does not take any arguments, e.g., {%% endif %%}")
		}
	case "else":
		info.Type = ControlElse
		if len(parts) > 1 {
			return nil, fmt.Errorf("else tag does not take any arguments, e.g., {%% else %%}")
		}
	case "elif":
		info.Type = ControlElseIf
		if len(parts) < 2 {
			return nil, fmt.Errorf("elif tag requires a condition, e.g., {%% elif user.isGuest %%}")
		}
		info.Expression = strings.Join(parts[1:], " ")
	case "for":
		info.Type = ControlFor
		// Parse "for item in items" pattern
		// Need at least 4 parts: "for", "item", "in", "items"
		if len(parts) < 4 {
			return nil, fmt.Errorf("for tag requires an item and collection, e.g., {%% for item in items %%}")
		}

		// First, check if there's a pipe filter in the expression
		// If there is, we need to handle it specially
		inPos := -1
		for i, part := range parts {
			if strings.ToLower(part) == "in" {
				inPos = i
				break
			}
		}

		if inPos == -1 {
			return nil, fmt.Errorf("for tag requires 'in' keyword, e.g., {%% for item in items %%}")
		}

		// Check for "for key, value in items" pattern (with or without filters)
		// This is a key-value unpacking loop
		if inPos >= 4 && parts[2] == "," {
			// The loop key variable is parts[1]
			// The loop value variable is parts[3]
			// The collection expression is everything after "in"
			keyVar := strings.TrimSpace(parts[1])
			valueVar := strings.TrimSpace(parts[3])
			collectionExpr := strings.Join(parts[inPos+1:], " ")

			info.Expression = fmt.Sprintf("%s, %s in %s", keyVar, valueVar, collectionExpr)
			return info, nil
		}

		// Otherwise, this is a standard item-in-collection loop
		// The loop variable is everything before the "in"
		// The collection expression is everything after "in"
		loopVar := strings.Join(parts[1:inPos], " ")
		collectionExpr := strings.Join(parts[inPos+1:], " ")

		info.Expression = fmt.Sprintf("%s in %s", loopVar, collectionExpr)
		return info, nil
	case "endfor":
		info.Type = ControlEndFor
		if len(parts) > 1 {
			return nil, fmt.Errorf("endfor tag does not take any arguments, e.g., {%% endfor %%}")
		}
	default:
		// For now, any unrecognized control tag keyword is marked as Unknown.
		// The raw content is stored in Expression for potential debugging or generic handling.
		info.Type = ControlUnknown
		info.Expression = trimmedContent // Store the original content if type is unknown
		// Depending on strictness, an error could be returned here:
		// return nil, fmt.Errorf("unknown control tag type: '%s'", tagTypeStr)
	}
	return info, nil
}

// parseControlTag is called when "{%" is found.
// It extracts the content between "{%" and "%}".
func (p *Parser) parseControlTag() *Node {
	originalPos := p.pos // For potential backtrack if parsing fails

	// Ensure we are actually at the start of a control tag
	if !(p.pos+2 <= len(p.input) && p.input[p.pos:p.pos+2] == "{%") {
		return nil // Not a control tag, or called incorrectly.
	}

	p.pos += 2                   // Consume "{%"
	controlContentStart := p.pos // The actual content starts after "{%"

	searchIndex := p.pos
	var controlContentEnd int = -1

	parenLevel := 0
	bracketLevel := 0
	braceLevel := 0

	for searchIndex < len(p.input) {
		// String literal skipping logic
		if p.input[searchIndex] == '\'' || p.input[searchIndex] == '"' {
			quoteChar := p.input[searchIndex]
			searchIndex++ // Move past the opening quote
			literalStringClosed := false
			stringContentScanStart := searchIndex
			for searchIndex < len(p.input) {
				if p.input[searchIndex] == quoteChar {
					isEscaped := false
					if searchIndex > stringContentScanStart {
						backslashCount := 0
						tempIdx := searchIndex - 1
						for tempIdx >= stringContentScanStart && p.input[tempIdx] == '\\' {
							backslashCount++
							tempIdx--
						}
						if backslashCount%2 == 1 {
							isEscaped = true
						}
					}
					if !isEscaped {
						searchIndex++ // Move past the closing quote
						literalStringClosed = true
						break
					}
				}
				searchIndex++
			}
			if !literalStringClosed {
				p.pos = originalPos // Backtrack
				// This indicates a syntax error within the control tag itself.
				// For simplicity, we might return a node that evaluation logic can flag as an error.
				// Or, the parser itself could signal an error node type or return an error.
				// For now, returning nil causes it to be treated as text.
				// A more robust parser might create an "ErrorNode" or allow ParseNext to return an error.
				return nil // Unclosed string literal within the control tag.
			}
			continue // Continue main scan for '%}'
		}

		// Check for nested tags first to avoid misinterpreting them as grouping symbols
		if searchIndex+1 < len(p.input) {
			if p.input[searchIndex] == '{' && (p.input[searchIndex+1] == '{' || p.input[searchIndex+1] == '%' || p.input[searchIndex+1] == '#') {
				// This is a nested Jinja tag, which is not allowed inside a control tag's logic.
				// However, for robust parsing, we should ideally handle it.
				// For now, we will let the grouping logic handle it, which might be incorrect for complex cases.
			}
		}

		// Track grouping levels to correctly find the end of the control tag
		switch p.input[searchIndex] {
		case '(':
			parenLevel++
		case ')':
			if parenLevel > 0 {
				parenLevel--
			}
		case '[':
			bracketLevel++
		case ']':
			if bracketLevel > 0 {
				bracketLevel--
			}
		case '{':
			braceLevel++
		case '}':
			if braceLevel > 0 {
				braceLevel--
			}
		}

		if searchIndex+1 < len(p.input) && p.input[searchIndex] == '%' && p.input[searchIndex+1] == '}' {
			if parenLevel == 0 && bracketLevel == 0 && braceLevel == 0 {
				controlContentEnd = searchIndex // Marks start of "%}"
				break                           // Found matching "%}"
			}
		}
		searchIndex++
	}

	if controlContentEnd == -1 {
		p.pos = originalPos // Backtrack
		return nil          // Indicate failure (unclosed control tag).
	}

	rawContent := p.input[controlContentStart:controlContentEnd]
	p.pos = controlContentEnd + 2 // Advance parser position past "%}"

	trimmedContent := strings.TrimSpace(rawContent)
	controlInfo, err := parseControlTagDetail(trimmedContent)
	if err != nil {
		// If parsing the detail fails (e.g. "if" without condition), we create a node
		// with ControlUnknown and the error message in Expression for easier debugging.
		// Alternatively, ParseNext could return this error.
		// For now, the node is created, and evaluation logic will see ControlUnknown or handle the error.
		// A simple approach is to make it an unknown tag with the original content.
		controlInfo = &ControlTagInfo{
			Type:       ControlUnknown,
			Expression: fmt.Sprintf("Error parsing tag '%s': %v", trimmedContent, err), // Store error for later
		}
	}

	// If parseControlTagDetail returns an error, controlInfo might be nil or partially filled.
	// For robustness, ensure controlInfo is not nil before creating the node,
	// or ensure parseControlTagDetail always returns a valid (even if "error" tagged) info.
	// The current parseControlTagDetail will return an error and a partially filled info for "unknown".
	// If it errors on valid tags like "if" without condition, it returns the error.
	// Let's ensure we always have a controlInfo, possibly marking it as parse_error.
	if controlInfo == nil && err != nil { // Should not happen if parseControlTagDetail is implemented carefully
		controlInfo = &ControlTagInfo{Type: ControlUnknown, Expression: fmt.Sprintf("Critical parsing error for: %s", trimmedContent)}
	}

	return &Node{
		Type:    NodeControlTag,
		Content: trimmedContent, // Store the trimmed raw content for reference
		Control: controlInfo,    // Store the parsed details
	}
}

// parseExpressionTag is called when "{{" is found.
// It extracts the content between "{{" and "}}".
// This remains largely the same, but the content it extracts will be processed by evaluateFullExpressionInternal.
func (p *Parser) parseExpressionTag() *Node {
	originalPos := p.pos // For potential backtrack if parsing fails

	// Ensure we are actually at the start of an expression tag
	if !(p.pos+2 <= len(p.input) && p.input[p.pos:p.pos+2] == "{{") {
		// Not an expression tag, or called incorrectly.
		return nil
	}

	p.pos += 2                      // Consume "{{"
	expressionContentStart := p.pos // The actual content starts after "{{"

	level := 1
	searchIndex := p.pos // Start scanning from here for content and end marker

	var expressionContentEnd int = -1

	parenLevel := 0
	bracketLevel := 0
	braceLevel := 0

	for searchIndex < len(p.input) {
		// String literal skipping logic
		if p.input[searchIndex] == '\'' || p.input[searchIndex] == '"' {
			quoteChar := p.input[searchIndex]
			searchIndex++ // Move past the opening quote
			literalStringClosed := false
			stringContentScanStart := searchIndex // for backslash check in string
			for searchIndex < len(p.input) {
				if p.input[searchIndex] == quoteChar {
					isEscaped := false
					// Check for escaped quote: count preceding backslashes
					if searchIndex > stringContentScanStart {
						backslashCount := 0
						tempIdx := searchIndex - 1
						// Count backslashes immediately preceding the quote
						for tempIdx >= stringContentScanStart && p.input[tempIdx] == '\\' {
							backslashCount++
							tempIdx--
						}
						if backslashCount%2 == 1 { // Odd number of backslashes means the quote is escaped
							isEscaped = true
						}
					}
					if !isEscaped {
						searchIndex++ // Move past the closing quote
						literalStringClosed = true
						break
					}
				}
				searchIndex++
			}
			if !literalStringClosed {
				// Unclosed string literal within the expression. This is a parse error for the tag.
				p.pos = originalPos // Backtrack
				return nil          // Signal failure
			}
			continue // Continue main scan for '{{' or '}}'
		}

		// Check for nested {{ and }} before handling grouping symbols
		if searchIndex+1 < len(p.input) {
			if p.input[searchIndex] == '{' && p.input[searchIndex+1] == '{' {
				level++
				searchIndex += 2
				continue
			} else if p.input[searchIndex] == '}' && p.input[searchIndex+1] == '}' {
				if level == 1 && parenLevel == 0 && bracketLevel == 0 && braceLevel == 0 {
					expressionContentEnd = searchIndex // Marks start of "}}"
					break                              // Found matching }}
				}
				level--
				searchIndex += 2
				continue
			}
		}

		// Track grouping levels
		switch p.input[searchIndex] {
		case '(':
			parenLevel++
		case ')':
			if parenLevel > 0 {
				parenLevel--
			}
		case '[':
			bracketLevel++
		case ']':
			if bracketLevel > 0 {
				bracketLevel--
			}
		case '{':
			braceLevel++
		case '}':
			if braceLevel > 0 {
				braceLevel--
			}
		}

		searchIndex++
	}

	if expressionContentEnd == -1 { // implies foundEndMarker is false / matching "}}" not found
		// Error: unclosed expression tag.
		p.pos = originalPos // Backtrack
		return nil          // Indicate failure.
	}

	// If we reach here, a matching "}}" was found.
	// The content is from expressionContentStart to expressionContentEnd.
	content := p.input[expressionContentStart:expressionContentEnd]
	p.pos = expressionContentEnd + 2 // Advance parser position past "}}"

	return &Node{
		Type:    NodeExpression,
		Content: content,
	}
}

// ParseNext returns the next node (text or expression) from the input stream.
// It returns (nil, nil) when EOF is reached.
// It relies on p.parseExpressionTag to handle the intricacies of expression parsing,
// including resetting p.pos if an expression tag is not properly closed.
func (p *Parser) ParseNext() (*Node, error) {
	if p.pos >= len(p.input) {
		return nil, nil // EOF, no error
	}

	// Check for comment marker "{#"
	if strings.HasPrefix(p.input[p.pos:], "{#") {
		commentNode := p.parseCommentTag()
		if commentNode != nil {
			return commentNode, nil
		}
		// If commentNode is nil, parseCommentTag failed (e.g. "#}" not found).
		// p.pos was reset by parseCommentTag.
		// Treat "{#" as literal text. Fall through.
	}

	// Check for control tag marker "{%"
	if strings.HasPrefix(p.input[p.pos:], "{%") {
		controlNode := p.parseControlTag()
		if controlNode != nil {
			return controlNode, nil
		}
		// If controlNode is nil, parseControlTag failed (e.g., "%}" not found).
		// p.pos was reset by parseControlTag.
		// Treat "{%" as literal text. Fall through.
	}

	// Check if current position starts with an expression marker "{{"
	if strings.HasPrefix(p.input[p.pos:], "{{") {
		// Attempt to parse it as a full expression tag
		// parseExpressionTag will advance p.pos on success, or reset p.pos on failure
		exprNode := p.parseExpressionTag()
		if exprNode != nil {
			// Successfully parsed an expression node
			return exprNode, nil
		}
		// If exprNode is nil, parseExpressionTag failed (e.g. "}}" not found).
		// p.pos was reset by parseExpressionTag.
		// In this case, the "{{" is treated as literal text.
		// We fall through to the text parsing logic below.
	}

	// Text parsing logic:
	// Find the next occurrence of "{#", "{{", "{%" or end of string.
	// This search starts from the current p.pos.
	nextCommentMarkerIndex := strings.Index(p.input[p.pos:], "{#")
	nextExprMarkerIndex := strings.Index(p.input[p.pos:], "{{")
	nextControlMarkerIndex := strings.Index(p.input[p.pos:], "{%")

	// Determine the earliest marker
	nextMarkerPos := -1

	if nextCommentMarkerIndex != -1 {
		nextMarkerPos = nextCommentMarkerIndex
	}

	if nextExprMarkerIndex != -1 {
		if nextMarkerPos == -1 || nextExprMarkerIndex < nextMarkerPos {
			nextMarkerPos = nextExprMarkerIndex
		}
	}

	if nextControlMarkerIndex != -1 {
		if nextMarkerPos == -1 || nextControlMarkerIndex < nextMarkerPos {
			nextMarkerPos = nextControlMarkerIndex
		}
	}

	if nextMarkerPos == -1 {
		// No more markers, the rest of the input is text.
		content := p.input[p.pos:]
		p.pos = len(p.input)   // Consume the rest of the input
		if len(content) == 0 { // Should only happen if called again after already at EOF
			return nil, nil
		}
		return &Node{Type: NodeText, Content: content}, nil
	}

	if nextMarkerPos == 0 {
		// This means p.input[p.pos:] starts with a marker AND its parsing failed above.
		// So, this specific marker (e.g. "{#", "{{") is literal.
		// The text node should include this marker and extend until the *next* different marker
		// that could start a new valid tag, or EOF.

		// We need to decide how much to consume as text. If it was "{{ an unclosed expr",
		// we should consume "{{".
		// Let's consume just the first character of the broken marker to ensure progress and re-evaluate.
		// Or, more robustly, find the *next earliest* different type of marker or actual next marker.

		// Search for the next marker of any type starting *after* the first character
		// of the current problematic marker to ensure progress.
		searchTextStartOffset := p.pos + 1

		if searchTextStartOffset >= len(p.input) { // e.g., input at p.pos is just "{" or "{#"
			content := p.input[p.pos:]
			p.pos = len(p.input)
			return &Node{Type: NodeText, Content: content}, nil
		}

		// Find the next occurrence of "{#" or "{{" starting from searchTextStartOffset
		nextNextCommentIdxRel := strings.Index(p.input[searchTextStartOffset:], "{#")
		nextNextExprIdxRel := strings.Index(p.input[searchTextStartOffset:], "{{")
		nextNextControlIdxRel := strings.Index(p.input[searchTextStartOffset:], "{%")

		nextNextMarkerAbs := -1

		if nextNextCommentIdxRel != -1 {
			currentAbs := searchTextStartOffset + nextNextCommentIdxRel
			if nextNextMarkerAbs == -1 || currentAbs < nextNextMarkerAbs {
				nextNextMarkerAbs = currentAbs
			}
		}
		if nextNextExprIdxRel != -1 {
			currentAbs := searchTextStartOffset + nextNextExprIdxRel
			if nextNextMarkerAbs == -1 || currentAbs < nextNextMarkerAbs {
				nextNextMarkerAbs = currentAbs
			}
		}
		if nextNextControlIdxRel != -1 {
			currentAbs := searchTextStartOffset + nextNextControlIdxRel
			if nextNextMarkerAbs == -1 || currentAbs < nextNextMarkerAbs {
				nextNextMarkerAbs = currentAbs
			}
		}

		if nextNextMarkerAbs == -1 {
			// No more markers found after the current problematic one.
			// The rest of the string from p.pos is literal text.
			content := p.input[p.pos:]
			p.pos = len(p.input)
			return &Node{Type: NodeText, Content: content}, nil
		}

		// Another marker was found. The text segment goes from p.pos up to this new marker.
		content := p.input[p.pos:nextNextMarkerAbs]
		p.pos = nextNextMarkerAbs // p.pos is now at the start of the *next* marker
		return &Node{Type: NodeText, Content: content}, nil

	} else { // nextMarkerPos > 0
		// Text exists before the next marker
		content := p.input[p.pos : p.pos+nextMarkerPos]
		p.pos += nextMarkerPos // Advance p.pos to the start of the next marker
		return &Node{Type: NodeText, Content: content}, nil
	}
}
