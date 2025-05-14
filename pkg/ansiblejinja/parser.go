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

		// Find matching "}}" for the "{{" at absoluteStartMarkerIndex
		// by counting brace levels.
		level := 1
		searchIndex := expressionContentStartIndex
		foundEndMarker := false
		absoluteEndMarkerIndex := -1 // Stores the index of the start of the matching "}}"

		for searchIndex < len(template) {
			if searchIndex+1 < len(template) {
				if template[searchIndex] == '{' && template[searchIndex+1] == '{' {
					level++
					searchIndex += 2 // Consume "{{"
					continue
				} else if template[searchIndex] == '}' && template[searchIndex+1] == '}' {
					level--
					if level == 0 {
						// Found the matching "}}"
						absoluteEndMarkerIndex = searchIndex
						foundEndMarker = true
						break
					}
					searchIndex += 2 // Consume "}}"
					continue
				}
			}
			searchIndex++
		}

		if !foundEndMarker {
			// No matching "}}" found, treat the rest from the original currentIndex as literal.
			// This handles cases like "Hello {{ name" or "{{ unclosed"
			segments = append(segments, segment{literalText, template[currentIndex:]})
			break
		} else {
			// Valid, matched tag found.
			// 1. Add literal text before "{{", if any.
			if absoluteStartMarkerIndex > currentIndex {
				segments = append(segments, segment{literalText, template[currentIndex:absoluteStartMarkerIndex]})
			}

			// 2. Add expression tag content.
			// Content is between expressionContentStartIndex and absoluteEndMarkerIndex.
			expressionContent := template[expressionContentStartIndex:absoluteEndMarkerIndex]
			segments = append(segments, segment{expressionTag, expressionContent})

			// 3. Update currentIndex to after "}}".
			currentIndex = absoluteEndMarkerIndex + 2
		}
	}

	return segments
}
