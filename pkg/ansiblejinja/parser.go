package ansiblejinja

import "strings"

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
