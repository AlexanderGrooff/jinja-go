package ansiblejinja

import (
	"fmt"
)

// This file will contain the logic for handling Jinja control statements
// such as {% if ... %}, {% for ... %}, etc.

// EvaluateExpressionFunc defines the signature for an expression evaluation function.
// This is used to pass EvaluateExpression logic to statement handlers.
type EvaluateExpressionFunc func(expression string, context map[string]interface{}) (interface{}, error)

// ProcessNodesFunc defines the signature for the node processing function.
// This is used to allow statement handlers to recursively process blocks of nodes.
type ProcessNodesFunc func(nodes []*Node, context map[string]interface{}) (string, error)

// handleIfStatement processes an If control tag and its corresponding block.
// It returns the rendered string for the block if the condition is true,
// the index of the node after the entire if/endif structure, and any error.
func handleIfStatement(
	nodes []*Node,
	currentIndex int,
	context map[string]interface{},
	evalExprFunc EvaluateExpressionFunc,
	processBlockNodesFunc ProcessNodesFunc,
) (renderedBlock string, nextIndex int, err error) {

	if currentIndex >= len(nodes) || nodes[currentIndex].Type != NodeControlTag || nodes[currentIndex].Control == nil || nodes[currentIndex].Control.Type != ControlIf {
		return "", currentIndex, fmt.Errorf("internal error: handleIfStatement called with non-if node at index %d", currentIndex)
	}

	// The overall structure spans from the initial {% if %} to its corresponding {% endif %}.
	// We need to find the ultimate endif first to know the bounds of the entire if-elif-else-endif construct.
	// findBlock with primaryStartType=ControlIf and primaryEndType=ControlEndIf, and no intermediate types, will find the matching endif.
	_, ultimateEndifIndex, findUltimateEndifErr := findBlock(nodes, currentIndex, ControlIf, ControlEndIf)
	if findUltimateEndifErr != nil {
		return "", currentIndex, findUltimateEndifErr // Unclosed if statement
	}

	// currentBranchStartIndex points to the start of the current if/elif tag being processed.
	currentBranchStartIndex := currentIndex
	conditionMet := false // Tracks if any if/elif condition has been met yet.
	output := ""

	// Loop through the branches: if, elif*, else?
	for currentBranchStartIndex < ultimateEndifIndex {
		branchNode := nodes[currentBranchStartIndex]
		if branchNode.Type != NodeControlTag || branchNode.Control == nil {
			// This shouldn't happen if parsing is correct and we are within an if structure.
			// It implies non-control-tag text/expr between if/elif/else branches, which is unusual.
			// For simplicity, let's treat it as an error or unexpected structure.
			return "", currentIndex, fmt.Errorf("unexpected node type '%v' within if/elif/else structure at index %d", branchNode.Type, currentBranchStartIndex)
		}

		branchType := branchNode.Control.Type
		branchExpression := branchNode.Control.Expression

		// Determine intermediate stoppers for findBlock
		var currentBranchIntermediateStoppers []ControlTagType
		if branchType == ControlIf || branchType == ControlElseIf {
			currentBranchIntermediateStoppers = []ControlTagType{ControlElseIf, ControlElse}
		} // For ControlElse, currentBranchIntermediateStoppers remains empty (nil)

		// Determine the end of the current branch's body.
		// This will be before the next elif/else or the final endif.
		bodyNodes, nextBranchOrEndifIndex, findBranchErr := findBlock(nodes, currentBranchStartIndex, branchType, ControlEndIf, currentBranchIntermediateStoppers...)
		if findBranchErr != nil {
			return "", currentIndex, findBranchErr
		}

		if branchType == ControlIf || branchType == ControlElseIf {
			if !conditionMet { // Only evaluate if no prior condition was met
				conditionResult, evalErr := evalExprFunc(branchExpression, context)
				if evalErr != nil {
					return "", currentIndex, fmt.Errorf("error evaluating condition for %s '%s': %v", branchType, branchExpression, evalErr)
				}
				truthy := isTruthy(conditionResult)

				if truthy {
					processedContent, err := processBlockNodesFunc(bodyNodes, context)
					if err != nil {
						return "", currentIndex, err
					}
					output = processedContent
					conditionMet = true
				}
			}
		} else if branchType == ControlElse {
			if !conditionMet { // Only execute else if no prior condition was met
				processedContent, err := processBlockNodesFunc(bodyNodes, context)
				if err != nil {
					return "", currentIndex, err
				}
				output = processedContent
				conditionMet = true // Even if else is empty, subsequent branches should not run.
			}
			// An else block, if executed, is the last conditional part.
			// The nextBranchOrEndifIndex should be the ultimate endif.
		} else if branchType == ControlEndIf {
			// This case should be handled by the outer loop's condition and ultimateEndifIndex check.
			// If we hit it here, it might mean findBlock logic needs refinement when called for if/elif/else.
			// For now, we assume findBlock correctly gives us the segment *before* the next clause or final endif.
			break // Should be caught by ultimateEndifIndex
		} else {
			return "", currentIndex, fmt.Errorf("unexpected control tag '%s' encountered within if structure at index %d", branchType, currentBranchStartIndex)
		}

		if conditionMet {
			// A condition has been met and its block processed (or determined to be empty).
			// We should now skip to the end of the entire if/elif/else/endif structure.
			return output, ultimateEndifIndex + 1, nil
		}

		// If condition was not met, move to the start of the next potential branch (elif/else) or the endif.
		currentBranchStartIndex = nextBranchOrEndifIndex

		// Before continuing the loop with the new currentBranchStartIndex,
		// check if we have landed on the ultimate endif or a valid next branch type.
		if currentBranchStartIndex < ultimateEndifIndex { // If not yet at the ultimate endif
			nextNode := nodes[currentBranchStartIndex]
			if nextNode.Type != NodeControlTag || nextNode.Control == nil {
				// This implies text or an expression where a control tag was expected (elif/else)
				return "", currentIndex, fmt.Errorf("unexpected text or expression found at node index %d, expected elif, else, or endif", currentBranchStartIndex)
			}

			nextTokenType := nextNode.Control.Type
			if nextTokenType == ControlUnknown {
				// A malformed tag (e.g. {% elif %}) was found where a valid elif/else was expected.
				return "", currentIndex, fmt.Errorf("malformed control tag '%s' found in if-structure: %s", nextNode.Content, nextNode.Control.Expression)
			}
			// Ensure it's a valid continuation or the end.
			if !(nextTokenType == ControlElseIf || nextTokenType == ControlElse || nextTokenType == ControlEndIf) {
				// Found something like {% for %} or another {% if %} where elif/else/endif was expected.
				return "", currentIndex, fmt.Errorf("unexpected tag type '%s' found at node index %d, expected elif, else, or endif", nextTokenType, currentBranchStartIndex)
			}
		} else if currentBranchStartIndex == ultimateEndifIndex {
			// We have advanced to the ultimate endif. Ensure it is indeed an endif tag.
			if nodes[currentBranchStartIndex].Type != NodeControlTag || nodes[currentBranchStartIndex].Control == nil || nodes[currentBranchStartIndex].Control.Type != ControlEndIf {
				return "", currentIndex, fmt.Errorf("expected endif tag at node index %d, but found %s", currentBranchStartIndex, nodes[currentBranchStartIndex].Control.Type)
			}
		} // If currentBranchStartIndex > ultimateEndifIndex, something went very wrong with findBlock.

		// If the nextBranchOrEndifIndex points to the ultimate endif, and no condition was met, the loop will terminate.
		if currentBranchStartIndex == ultimateEndifIndex && nodes[currentBranchStartIndex].Control.Type == ControlEndIf {
			break
		}
	}

	// If we exit the loop, it means we've processed all branches or reached the endif without meeting a condition.
	return output, ultimateEndifIndex + 1, nil
}

// findBlock locates the nodes within a control block (e.g., if...endif) and the index of the closing tag.
// It handles nested blocks of the same type (e.g., nested ifs).
// For if-blocks, it returns nodes between {% if %} and the next {% else %}, {% elif %}, or {% endif %}.
func findBlock(nodes []*Node, startIndex int, primaryStartType ControlTagType, primaryEndType ControlTagType, intermediateTypes ...ControlTagType) (blockNodes []*Node, closingTagIndex int, err error) {
	if startIndex >= len(nodes) || nodes[startIndex].Type != NodeControlTag || nodes[startIndex].Control == nil || nodes[startIndex].Control.Type != primaryStartType {
		return nil, -1, fmt.Errorf("internal error: findBlock called with incorrect start node at index %d. Expected type %s, got %s (content: '%s')", startIndex, primaryStartType, nodes[startIndex].Control.Type, nodes[startIndex].Content)
	}

	nestingLevel := 1 // Starts at 1 because nodes[startIndex] is the block opener.

	for i := startIndex + 1; i < len(nodes); i++ {
		node := nodes[i]
		if node.Type == NodeControlTag && node.Control != nil {
			tagType := node.Control.Type
			originalNestingLevelBeforeThisTag := nestingLevel

			// Step 1: Adjust nesting based on generic block delimiters (if/endif)
			// Future: Extend this for other nestable tags like for/endfor.
			if tagType == ControlIf {
				nestingLevel++
			} else if tagType == ControlEndIf {
				nestingLevel--
			}

			// Step 2: Check for segment termination based on intermediate types or unknown tags
			// This check is relevant if the block was at nesting level 1 *before* this tag was processed.
			if originalNestingLevelBeforeThisTag == 1 {
				isIntermediateStopper := false
				for _, itype := range intermediateTypes {
					if tagType == itype {
						isIntermediateStopper = true
						break
					}
				}
				if isIntermediateStopper {
					// Found an intermediate tag (e.g., 'else' for an 'if' block).
					// This terminates the current segment.
					return nodes[startIndex+1 : i], i, nil
				}

				if tagType == ControlUnknown {
					// A malformed tag at level 1 also terminates the current segment.
					return nodes[startIndex+1 : i], i, nil
				}
			}

			// Step 3: Check if the primary block has definitively closed.
			// This happens when nestingLevel becomes 0, and the tag causing it is the primaryEndType.
			if nestingLevel == 0 {
				if tagType == primaryEndType {
					// The block defined by primaryStartType and primaryEndType is now closed.
					return nodes[startIndex+1 : i], i, nil
				}
				// If nestingLevel is 0 but tagType is not primaryEndType,
				// it implies a mismatched tag structure (e.g., {% if %}{% for %}{% endif %}).
				// This will eventually lead to an "unclosed primaryStartType" error, which is appropriate.
			}

			if nestingLevel < 0 {
				// Too many closing tags encountered relative to opening ones.
				// This indicates a malformed structure.
				// The "unclosed primaryStartType" error at the end of the function will catch this,
				// as the primaryEndType will not be found correctly.
				// Alternatively, could return a specific error here:
				// return nil, -1, fmt.Errorf("mismatched closing tag '%s' at index %d, nesting level became %d", tagType, i, nestingLevel)
			}
		}
		// Continue scanning if not a relevant control tag or if nested deeper and not yet resolved.
	}

	// If loop finishes and nestingLevel is still > 0, the block was not properly closed.
	return nil, -1, fmt.Errorf("unclosed '%s' tag starting at node index %d (content: '%s')", nodes[startIndex].Control.Type, startIndex, nodes[startIndex].Content)
}
