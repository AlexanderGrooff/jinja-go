package ansiblejinja

import (
	"fmt"
	"strconv"
	"strings"
)

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

// Node represents a parsed element in the template, such as literal text or an expression.
type Node struct {
	Type    NodeType // The kind of node (e.g., text, expression).
	Content string   // The raw content of the node.
	// Future additions: original start/end positions, evaluated value for expressions, etc.
}

// NodeType defines the category of a parsed Node.
type NodeType int

// Enumerates the different types of nodes that can be encountered during parsing.
const (
	NodeText       NodeType = iota // Represents a segment of literal text.
	NodeExpression                 // Represents a Jinja expression, e.g., {{ variable }}.
	// NodeComment             // Future: Represents a comment, e.g., {# comment #}.
	// NodeTag                 // Future: Represents a control structure tag, e.g., {% if ... %}.
)

// segmentType defines the type of a parsed segment.
type segmentType int

const (
	// literalText represents a segment of plain text.
	literalText segmentType = iota
	// expressionTag represents a Jinja expression (e.g., {{ variable }}).
	expressionTag
)

// segment represents a piece of the parsed template string.
type segment struct {
	segmentType segmentType
	content     string
}

// parseTemplate tokenizes the template string into a slice of segments,
// distinguishing between literal text and expressions.
// For example, "Hello {{ name }}!" would be parsed into:
//   - segment{literalText, "Hello "}
//   - segment{expressionTag, " name "}
//   - segment{literalText, "!"}
//
// If an opening "{{" is not matched by a closing "}}", the "{{" and
// the text following it (until the end of the string or the next valid marker)
// are treated as literal text.
func parseTemplate(template string) []segment {
	var segments []segment
	currentIndex := 0

	for currentIndex < len(template) {
		startMarkerRel := strings.Index(template[currentIndex:], "{{")

		if startMarkerRel == -1 {
			segments = append(segments, segment{literalText, template[currentIndex:]})
			break
		}

		absoluteStartMarkerIndex := currentIndex + startMarkerRel
		expressionContentStartIndex := absoluteStartMarkerIndex + 2

		level := 1
		searchIndex := expressionContentStartIndex
		foundEndMarker := false
		var absoluteEndMarkerIndex int = -1

		for searchIndex < len(template) {
			// Check for string literals first to ensure '{{' or '}}' inside a string are skipped for level counting
			if template[searchIndex] == '\'' || template[searchIndex] == '"' {
				quoteChar := template[searchIndex]
				searchIndex++ // Move past the opening quote

				stringContentStart := searchIndex
				literalStringClosed := false
				for searchIndex < len(template) {
					if template[searchIndex] == quoteChar {
						// Check for escaped quote: count preceding backslashes
						isEscaped := false
						if searchIndex > stringContentStart { // Ensure there's a character before to check for backslash
							backslashCount := 0
							tempIdx := searchIndex - 1
							for tempIdx >= stringContentStart && template[tempIdx] == '\\' { // Correctly escaped backslash for comparison
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
					// Unclosed string literal within the expression.
					// This will lead to foundEndMarker remaining false, which is handled later.
					break // Exit the main scanning loop for this expression
				}
				// After processing a string, continue to the next char in the main expression scan
				continue
			}

			// Check for {{ and }} if not in a string literal processing
			if searchIndex+1 < len(template) {
				if template[searchIndex] == '{' && template[searchIndex+1] == '{' {
					level++
					searchIndex += 2
					continue
				} else if template[searchIndex] == '}' && template[searchIndex+1] == '}' {
					level--
					if level == 0 {
						absoluteEndMarkerIndex = searchIndex
						foundEndMarker = true
						break // Found matching }}
					}
					searchIndex += 2
					continue
				}
			}

			// If none of the above, just move to the next character in the expression
			searchIndex++
		}

		if !foundEndMarker {
			segments = append(segments, segment{literalText, template[currentIndex:]})
			break
		} else {
			if absoluteStartMarkerIndex > currentIndex {
				segments = append(segments, segment{literalText, template[currentIndex:absoluteStartMarkerIndex]})
			}

			expressionContent := template[expressionContentStartIndex:absoluteEndMarkerIndex]
			segments = append(segments, segment{expressionTag, expressionContent})

			currentIndex = absoluteEndMarkerIndex + 2
		}
	}

	return segments
}

// evaluateFullExpressionInternal is the core logic for evaluating an expression string,
// potentially including filters.
// It returns the final value, a boolean indicating if the base variable was strictly undefined
// (and not resolved by a filter like default), and an error if parsing/evaluation failed.
func evaluateFullExpressionInternal(fullExprStr string, context map[string]interface{}) (value interface{}, wasStrictlyUndefined bool, err error) {
	parts := splitExpressionWithFilters(fullExprStr)
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" && len(parts) == 1 && fullExprStr != "" && strings.TrimSpace(fullExprStr) != "" {
		// Handles "{{}}" or "{{   }}" resulting in empty base expression vs an explicitly empty key like "{{ '' | ... }}"
		// If fullExprStr itself was non-empty but trimmed to empty for parts[0] with no filters, it might be "{{ }}".
		// In Jinja, {{}} or {{ }} often means lookup of empty string key or results in empty string.
		// Let's treat a completely empty or whitespace-only expression (after trimming node.content) as lookup for ""
		if strings.TrimSpace(fullExprStr) == "" {
			baseExprToLookup := ""
			val, exists := context[baseExprToLookup]
			if !exists {
				return nil, true, nil // Undefined, no error for TemplateString to make empty
			}
			return val, false, nil
		}
		// If parts[0] is empty but there are filters, e.g. "{{ | default('empty') }}" - this is likely a syntax error.
		// Or if fullExprStr had content that split into an empty first part.
		return nil, false, fmt.Errorf("empty base expression before filter pipeline: '%s'", fullExprStr)
	}

	baseExpr := strings.TrimSpace(parts[0])
	var currentValue interface{}
	initialLookupFailed := false

	// 1. Evaluate the base expression (literal or variable)
	if (strings.HasPrefix(baseExpr, "'") && strings.HasSuffix(baseExpr, "'")) ||
		(strings.HasPrefix(baseExpr, "\"") && strings.HasSuffix(baseExpr, "\"")) {
		if len(baseExpr) >= 2 {
			currentValue = unescapeString(baseExpr[1 : len(baseExpr)-1])
		} else {
			currentValue = "" // Should not happen if prefix/suffix match
		}
	} else if i, errConv := strconv.Atoi(baseExpr); errConv == nil {
		currentValue = i
	} else if bVal, errConv := strconv.ParseBool(baseExpr); errConv == nil {
		currentValue = bVal
	} else {
		// Not a recognized literal. Assume it's a variable name.
		if !isValidJinjaIdentifier(baseExpr) {
			return nil, false, fmt.Errorf("invalid syntax for expression or variable name: '%s'", baseExpr)
		}
		val, exists := context[baseExpr]
		if !exists {
			initialLookupFailed = true
			currentValue = nil // Represents undefined for now
		} else {
			currentValue = val
		}
	}

	currentValueIsEffectivelyUndefined := initialLookupFailed

	// 2. Apply filters
	for i := 1; i < len(parts); i++ {
		filterCallStr := strings.TrimSpace(parts[i])
		if filterCallStr == "" {
			return nil, false, fmt.Errorf("empty filter declaration in pipeline: '%s'", fullExprStr)
		}
		filterName, filterArgsRaw, errParseFilter := parseFilterCall(filterCallStr)
		if errParseFilter != nil {
			return nil, false, fmt.Errorf("failed to parse filter call '%s': %v", filterCallStr, errParseFilter)
		}

		filterFunc, ok := GlobalFilters[filterName]
		if !ok {
			return nil, false, fmt.Errorf("unknown filter '%s'", filterName)
		}

		var evaluatedArgs []interface{}
		for _, argStr := range filterArgsRaw {
			evalArg, errEvalArg := evaluateLiteralOrVariable(argStr, context)
			if errEvalArg != nil {
				return nil, false, fmt.Errorf("failed to evaluate filter argument '%s' for filter '%s': %v", argStr, filterName, errEvalArg)
			}
			evaluatedArgs = append(evaluatedArgs, evalArg)
		}

		var filterErr error
		// The `currentValue` (which might be nil if initialLookupFailed) is passed to the filter.
		// The filter (e.g., `defaultFilter`) is responsible for handling this based on its logic.
		currentValue, filterErr = filterFunc(currentValue, evaluatedArgs...)
		if filterErr != nil {
			return nil, false, fmt.Errorf("error applying filter '%s': %v", filterName, filterErr)
		}

		// If a filter (like 'default') was applied to an initially undefined variable
		// and produced a result, the value is no longer "effectively undefined" for the rest of the pipeline
		// or for the final "strict undefined" check.
		if initialLookupFailed { // Check if the *original* variable was the source of undefinedness
			// If default filter (or similar) just ran, it might have resolved the undefined state.
			// We mark it as no longer "effectively undefined" if currentValue is now non-nil,
			// or if the filter is known to handle undefined (like 'default').
			// A simpler check: if defaultFilter was called on an undefined var, it 'handles' it.
			if filterName == "default" {
				currentValueIsEffectivelyUndefined = false
			}
			// A more general rule: if input was nil due to lookup failure, and filter returns non-nil, it's "resolved".
			// However, `default` specifically makes it "not undefined" for strict checks.
		}
	}
	return currentValue, initialLookupFailed && currentValueIsEffectivelyUndefined, nil
}

// evaluateLiteralOrVariable tries to parse a string as a literal (string, int, bool)
// or looks it up as a variable in the context.
func evaluateLiteralOrVariable(s string, context map[string]interface{}) (interface{}, error) {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return nil, fmt.Errorf("empty string cannot be evaluated as a literal or variable for filter argument")
	}

	// Check for string literal (single or double quotes)
	if (strings.HasPrefix(trimmed, "'") && strings.HasSuffix(trimmed, "'")) ||
		(strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"")) {
		if len(trimmed) >= 2 {
			return unescapeString(trimmed[1 : len(trimmed)-1]), nil
		}
		return "", nil // Should ideally not happen if prefix/suffix matched
	}

	// Check for boolean
	if trimmed == "true" {
		return true, nil
	}
	if trimmed == "false" {
		return false, nil
	}

	// Check for integer
	if i, err := strconv.Atoi(trimmed); err == nil {
		return i, nil
	}

	// Check for float (Jinja supports float literals e.g. 3.14)
	if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return f, nil
	}

	// Assume it's a variable name if not a recognized literal
	if !isValidJinjaIdentifier(trimmed) {
		// This case means it's not a simple literal, and not a valid var name.
		// e.g. an arg like "foo bar" (unquoted) or "foo[bar"
		return nil, fmt.Errorf("filter argument '%s' is not a valid literal or variable name", trimmed)
	}

	val, exists := context[trimmed]
	if !exists {
		return nil, fmt.Errorf("variable '%s' used as filter argument not found in context", trimmed)
	}
	return val, nil
}

// unescapeString handles basic unescaping for string literals.
func unescapeString(s string) string {
	s = strings.ReplaceAll(s, "\\\\'", "'")       // Escaped single quote
	s = strings.ReplaceAll(s, "\\\\\\\"", "\"")   // Escaped double quote
	s = strings.ReplaceAll(s, "\\\\\\\\", "\\\\") // Escaped backslash
	// TODO: Add more common escapes like \\n, \\t if necessary
	return s
}

// isValidJinjaIdentifier checks if a string is a valid Jinja identifier (simplified).
// It's used for variable names and filter arguments that are variables.
func isValidJinjaIdentifier(name string) bool {
	if name == "" {
		return false // An empty string is not a valid identifier.
	}
	// Crude check: disallow whitespace and characters used in Jinja syntax for pipes, calls, literals.
	// Jinja identifiers are more like Python: start with letter or underscore, then letters, numbers, underscores.
	// This is a much simpler check for now.
	// The set of characters to disallow in an identifier:
	// space, tab, newline, CR, {, }, |, ", ', (, ), ,, [, ]
	disallowedChars := " \t\n\r{{}}|'\"(),[]" // Note: " is escaped as \"
	if strings.ContainsAny(name, disallowedChars) {
		return false
	}
	// Optionally, check if it starts with a digit if that's not allowed for your var names
	// (unless it's purely a number, which is handled by literal parsing already).
	// For now, if it passes ContainsAny, it's "valid enough" for this simplified model.
	return true
}

// splitExpressionWithFilters splits an expression string by the pipe '|'
// ensuring that pipes within string literals or parentheses are ignored. This is a simplified version.
func splitExpressionWithFilters(fullExprStr string) []string {
	var parts []string
	var currentPart strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	parenLevel := 0 // To ignore pipes within parentheses, e.g. a_func(b | c)

	for i, r := range fullExprStr {
		switch r {
		case '\'':
			if i > 0 && fullExprStr[i-1] != '\\' { // Not an escaped backslash before quote
				inSingleQuote = !inSingleQuote
			}
			currentPart.WriteRune(r)
		case '"':
			if i > 0 && fullExprStr[i-1] != '\\' {
				inDoubleQuote = !inDoubleQuote
			}
			currentPart.WriteRune(r)
		case '(':
			if !inSingleQuote && !inDoubleQuote {
				parenLevel++
			}
			currentPart.WriteRune(r)
		case ')':
			if !inSingleQuote && !inDoubleQuote {
				if parenLevel > 0 { // Ensure we don't go negative if unbalanced
					parenLevel--
				}
			}
			currentPart.WriteRune(r)
		case '|':
			if !inSingleQuote && !inDoubleQuote && parenLevel == 0 {
				parts = append(parts, strings.TrimSpace(currentPart.String()))
				currentPart.Reset()
			} else {
				currentPart.WriteRune(r)
			}
		default:
			currentPart.WriteRune(r)
		}
	}
	parts = append(parts, strings.TrimSpace(currentPart.String()))

	// Filter out any truly empty parts that might result from multiple pipes, e.g., "var || filter"
	// or leading/trailing pipes if not handled by TrimSpace on currentPart effectively enough.
	finalParts := []string{}
	for _, p := range parts {
		if p != "" { // Only add non-empty parts
			finalParts = append(finalParts, p)
		} else if len(finalParts) == 0 && len(parts) > 1 {
			// Handle cases like "{{ | default('foo') }}" -> parts might be ["", "default('foo')"]
			// The first part being empty is handled in evaluateFullExpressionInternal
			finalParts = append(finalParts, p) // Keep the leading empty part if it's intentional.
		}
	}
	if len(finalParts) == 0 && fullExprStr != "" && strings.TrimSpace(fullExprStr) != "" {
		// If fullExprStr was e.g. "   " and split into nothing, but evaluateFullExpressionInternal expects baseExpr.
		// This can happen if fullExprStr is just whitespace. split gives [""] or [].
		// evaluateFullExpressionInternal handles empty baseExpr.
		return []string{strings.TrimSpace(fullExprStr)} // Ensure at least the trimmed original if it was all spaces
	}

	return finalParts
}

// parseFilterCall parses a filter string like "default('fallback', true)"
// into filter name "default" and raw (un-evaluated) arguments ["'fallback'", "true"].
func parseFilterCall(filterCallStr string) (name string, args []string, err error) {
	filterCallStr = strings.TrimSpace(filterCallStr)
	openParen := strings.Index(filterCallStr, "(")

	if openParen == -1 { // Filter without arguments, e.g., {{ my_var | upper }}
		if !isValidJinjaIdentifier(filterCallStr) { // Validate filter name itself
			return "", nil, fmt.Errorf("invalid filter name: '%s'", filterCallStr)
		}
		return filterCallStr, nil, nil
	}

	// Ensure the character before '(' is not part of a valid identifier, if '(' is not at the start
	// This is to prevent misinterpreting something like "func(arg)" as filter "func(arg)".
	// Filter names should be simple identifiers.
	filterNameCandidate := strings.TrimSpace(filterCallStr[:openParen])
	if !isValidJinjaIdentifier(filterNameCandidate) {
		return "", nil, fmt.Errorf("invalid filter name format before '(': '%s'", filterNameCandidate)
	}
	name = filterNameCandidate

	if !strings.HasSuffix(filterCallStr, ")") {
		return "", nil, fmt.Errorf("filter call with arguments missing closing parenthesis: '%s'", filterCallStr)
	}
	// closeParen := strings.LastIndex(filterCallStr, ")")
	// if closeParen != len(filterCallStr)-1 { // Defensive, already checked by HasSuffix
	// 	return "", nil, fmt.Errorf("mismatched parentheses or trailing characters in filter call: '%s'", filterCallStr)
	// }

	argsStr := filterCallStr[openParen+1 : len(filterCallStr)-1] // Content between ()

	if strings.TrimSpace(argsStr) == "" { // Handles "filter()" - no arguments
		return name, []string{}, nil
	}

	// Argument parsing: This is complex. For "name(arg1, 'arg2, still arg2', arg3)"
	// A simple strings.Split by ',' will fail.
	// Need to respect quotes and potentially nested structures.
	var parsedArgs []string
	var currentArg strings.Builder
	argInSingleQuote := false
	argInDoubleQuote := false
	argParenLevel := 0

	for i, r := range argsStr {
		switch r {
		case '\'':
			if i > 0 && argsStr[i-1] != '\\' {
				argInSingleQuote = !argInSingleQuote
			}
			currentArg.WriteRune(r)
		case '"':
			if i > 0 && argsStr[i-1] != '\\' {
				argInDoubleQuote = !argInDoubleQuote
			}
			currentArg.WriteRune(r)
		case '(':
			if !argInSingleQuote && !argInDoubleQuote {
				argParenLevel++
			}
			currentArg.WriteRune(r)
		case ')':
			if !argInSingleQuote && !argInDoubleQuote {
				if argParenLevel > 0 {
					argParenLevel--
				}
			}
			currentArg.WriteRune(r)
		case ',':
			if !argInSingleQuote && !argInDoubleQuote && argParenLevel == 0 {
				parsedArgs = append(parsedArgs, strings.TrimSpace(currentArg.String()))
				currentArg.Reset()
			} else {
				currentArg.WriteRune(r)
			}
		default:
			currentArg.WriteRune(r)
		}
	}
	// Add the last argument
	if currentArg.Len() > 0 || len(parsedArgs) == 0 && strings.TrimSpace(argsStr) != "" {
		// Add if currentArg has content, OR if there are no parsedArgs yet but argsStr was not empty (single arg case)
		lastArg := strings.TrimSpace(currentArg.String())
		if lastArg != "" || (len(parsedArgs) == 0 && strings.TrimSpace(argsStr) != "") {
			parsedArgs = append(parsedArgs, lastArg)
		}
	}

	// Filter out any empty strings that might result from parsing ", ," or trailing commas, unless it's a single empty string literal.
	// Example: default('') should yield one arg: "''"
	// default(a,,b) should yield "a", "b" (middle one skipped if truly empty after trim)
	finalArgs := []string{}
	for _, arg := range parsedArgs {
		trimmedArg := strings.TrimSpace(arg)
		if trimmedArg != "" { // Only add non-empty args after trim
			finalArgs = append(finalArgs, trimmedArg)
		} else if (strings.HasPrefix(arg, "'") && strings.HasSuffix(arg, "'") && len(arg) == 2) ||
			(strings.HasPrefix(arg, "\"") && strings.HasSuffix(arg, "\"") && len(arg) == 2) {
			// It's an explicit empty string literal like '' or ""
			finalArgs = append(finalArgs, arg)
		}
	}

	return name, finalArgs, nil
}

// parseTag is called when "{{" is found.
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

		// Check for nested {{ and }}
		if searchIndex+1 < len(p.input) {
			if p.input[searchIndex] == '{' && p.input[searchIndex+1] == '{' {
				level++
				searchIndex += 2
				continue
			} else if p.input[searchIndex] == '}' && p.input[searchIndex+1] == '}' {
				level--
				if level == 0 {
					expressionContentEnd = searchIndex // Marks start of "}}"
					break                              // Found matching }}
				}
				searchIndex += 2
				continue
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

	// Assuming Node and NodeExpression (a NodeType constant) are defined
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
	// Find the next occurrence of "{{" or end of string.
	// This search starts from the current p.pos.
	nextMarkerIndexInSubstring := strings.Index(p.input[p.pos:], "{{")

	if nextMarkerIndexInSubstring == -1 {
		// No more "{{" markers, the rest of the input is text.
		content := p.input[p.pos:]
		p.pos = len(p.input)   // Consume the rest of the input
		if len(content) == 0 { // Should only happen if called again after already at EOF
			return nil, nil
		}
		return &Node{Type: NodeText, Content: content}, nil
	}

	if nextMarkerIndexInSubstring == 0 {
		// This means p.input[p.pos:] starts with "{{" AND parseExpressionTag (called above) failed for it.
		// So, this specific "{{" is literal. The text node should include this "{{"
		// and extend until the *next* "{{" that could start a new valid expression, or EOF.

		// Search for the next "{{" starting *after* the first character of the current problematic "{{"
		// to ensure progress.
		searchTextStartOffset := p.pos + 1 // Start search after the initial '{'

		// Handle edge cases where input is very short (e.g., just "{" or "{{")
		if searchTextStartOffset > len(p.input) { // e.g., input at p.pos is just "{"
			content := p.input[p.pos:]
			p.pos = len(p.input)
			return &Node{Type: NodeText, Content: content}, nil
		}

		// If p.input[p.pos:] was "{{" and parseExpressionTag failed, searchTextStartOffset is p.pos + 1.
		// input[searchTextStartOffset:] is the second "{". Index of "{{" in "{" is -1.
		nextNextMarkerRelIndex := strings.Index(p.input[searchTextStartOffset:], "{{")

		if nextNextMarkerRelIndex == -1 {
			// No more "{{" found after the current problematic one.
			// The rest of the string from p.pos is literal text.
			content := p.input[p.pos:]
			p.pos = len(p.input)
			return &Node{Type: NodeText, Content: content}, nil
		}

		// Another "{{" was found. The text segment goes from p.pos up to this new "{{"
		// nextNextMarkerRelIndex is relative to searchTextStartOffset.
		// Absolute end of the text segment is searchTextStartOffset + nextNextMarkerRelIndex.
		endOfTextAbs := searchTextStartOffset + nextNextMarkerRelIndex
		content := p.input[p.pos:endOfTextAbs]
		p.pos = endOfTextAbs // p.pos is now at the start of the *next* "{{"
		return &Node{Type: NodeText, Content: content}, nil

	} else { // nextMarkerIndexInSubstring > 0
		// Text exists before the next "{{"
		content := p.input[p.pos : p.pos+nextMarkerIndexInSubstring]
		p.pos += nextMarkerIndexInSubstring // Advance p.pos to the start of the next "{{"
		return &Node{Type: NodeText, Content: content}, nil
	}
}
