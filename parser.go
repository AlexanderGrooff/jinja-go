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
	// NodeTag                 // Future: Represents a control structure tag, e.g., {% if ... %}.
)

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

		// Check for dot notation (e.g., loop.index, user.name)
		if strings.Contains(baseExpr, ".") && !strings.Contains(baseExpr, " ") {
			// Try to evaluate as dot notation
			dotVal, err := evaluateDotNotation(baseExpr, context)
			if err == nil {
				currentValue = dotVal
			} else {
				initialLookupFailed = true
				currentValue = nil // Represents undefined for now
			}
		} else {
			// Regular variable lookup
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
	s = strings.ReplaceAll(s, "\\'", "'")   // Escaped single quote
	s = strings.ReplaceAll(s, "\\\"", "\"") // Escaped double quote
	s = strings.ReplaceAll(s, "\\\\", "\\") // Escaped backslash
	s = strings.ReplaceAll(s, "\\n", "\n")  // Escaped newline
	s = strings.ReplaceAll(s, "\\t", "\t")  // Escaped tab
	s = strings.ReplaceAll(s, "\\r", "\r")  // Escaped carriage return
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
			// Check if this is an escaped quote
			isEscaped := false
			if i > 0 && argsStr[i-1] == '\\' {
				// Check if the backslash itself is escaped
				if i > 1 && argsStr[i-2] == '\\' {
					// The backslash was escaped, so the quote is not escaped
					isEscaped = false
				} else {
					// The quote is escaped
					isEscaped = true
				}
			}

			if !isEscaped {
				argInSingleQuote = !argInSingleQuote
			}
			currentArg.WriteRune(r)
		case '"':
			// Check if this is an escaped quote
			isEscaped := false
			if i > 0 && argsStr[i-1] == '\\' {
				// Check if the backslash itself is escaped
				if i > 1 && argsStr[i-2] == '\\' {
					// The backslash was escaped, so the quote is not escaped
					isEscaped = false
				} else {
					// The quote is escaped
					isEscaped = true
				}
			}

			if !isEscaped {
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

		if searchIndex+1 < len(p.input) && p.input[searchIndex] == '%' && p.input[searchIndex+1] == '}' {
			controlContentEnd = searchIndex // Marks start of "%}"
			break                           // Found matching "%}"
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
