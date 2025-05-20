package ansiblejinja

import (
	"fmt"
	"reflect"
	"strings"
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
			} else if tagType == ControlFor {
				nestingLevel++
			} else if tagType == ControlEndFor {
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

// handleForStatement processes a For control tag and its corresponding block.
// It returns the rendered string for the entire for loop, the index of the node after
// the entire for/endfor structure, and any error.
func handleForStatement(
	nodes []*Node,
	currentIndex int,
	context map[string]interface{},
	evalExprFunc EvaluateExpressionFunc,
	processBlockNodesFunc ProcessNodesFunc,
) (renderedBlock string, nextIndex int, err error) {
	if currentIndex >= len(nodes) || nodes[currentIndex].Type != NodeControlTag ||
		nodes[currentIndex].Control == nil || nodes[currentIndex].Control.Type != ControlFor {
		return "", currentIndex, fmt.Errorf("internal error: handleForStatement called with non-for node at index %d", currentIndex)
	}

	// Find the matching endfor tag
	bodyNodes, endForIndex, findBlockErr := findBlock(nodes, currentIndex, ControlFor, ControlEndFor)
	if findBlockErr != nil {
		return "", currentIndex, findBlockErr // Unclosed for statement
	}

	// Parse "item in items" expression
	forNode := nodes[currentIndex]
	forExpr := forNode.Control.Expression

	// The expression should be in format "item in items"
	parts := strings.SplitN(forExpr, " in ", 2)
	if len(parts) != 2 {
		return "", currentIndex, fmt.Errorf("invalid for loop expression: %s", forExpr)
	}

	loopVarName := strings.TrimSpace(parts[0])
	collectionExpr := strings.TrimSpace(parts[1])

	// Evaluate the collection expression
	var collectionVal interface{}
	var evalErr error

	// First try to evaluate as a compound expression for nested properties
	collectionVal, evalErr = EvaluateCompoundExpression(collectionExpr, context)
	if evalErr != nil {
		// Fall back to simple evaluation
		collectionVal, evalErr = evalExprFunc(collectionExpr, context)
		if evalErr != nil {
			return "", currentIndex, fmt.Errorf("error evaluating for loop collection '%s': %v", collectionExpr, evalErr)
		}
	}

	// Convert the collection to a slice for iteration
	collection, err := convertToSlice(collectionVal)
	if err != nil {
		return "", currentIndex, fmt.Errorf("for loop requires an iterable collection: %v", err)
	}

	// Prepare the results from iterating over the collection
	var result strings.Builder

	// Create loop context for each iteration
	for i, item := range collection {
		// Create a copy of the context for this iteration
		iterContext := make(map[string]interface{})
		for k, v := range context {
			iterContext[k] = v
		}

		// Add the loop variable to the context
		iterContext[loopVarName] = item

		// Add the 'loop' special variable with iteration information
		// Using integers for numeric values to ensure proper comparisons
		loopInfo := map[string]interface{}{
			"index":     i + 1,                   // 1-based index
			"index0":    i,                       // 0-based index
			"first":     i == 0,                  // True if first iteration
			"last":      i == len(collection)-1,  // True if last iteration
			"length":    len(collection),         // Total number of items
			"revindex":  len(collection) - i,     // Reverse index (1-based)
			"revindex0": len(collection) - i - 1, // Reverse index (0-based)
		}
		iterContext["loop"] = loopInfo

		// Declare the function variable first to allow for recursion
		var customProcessNodesFunc ProcessNodesFunc

		// Define the function implementation
		customProcessNodesFunc = func(nodes []*Node, loopContext map[string]interface{}) (string, error) {
			var sb strings.Builder
			nodeIndex := 0

			for nodeIndex < len(nodes) {
				node := nodes[nodeIndex]
				switch node.Type {
				case NodeExpression:
					// Special handling for expressions with dot notation in for loops
					if strings.Contains(node.Content, ".") {
						// Check for loop.index and other loop variables
						trimmedContent := strings.TrimSpace(node.Content)
						if strings.HasPrefix(trimmedContent, "loop.") {
							loopObj, ok := loopContext["loop"].(map[string]interface{})
							if ok {
								parts := strings.SplitN(trimmedContent, ".", 2)
								if len(parts) == 2 {
									attrName := parts[1]
									if val, exists := loopObj[attrName]; exists {
										sb.WriteString(fmt.Sprintf("%v", val))
										nodeIndex++
										continue
									}
								}
							}
						}

						// Check for item.attribute pattern
						if strings.Contains(trimmedContent, ".") && !strings.Contains(trimmedContent, " ") {
							parts := strings.SplitN(trimmedContent, ".", 2)
							if len(parts) == 2 {
								obj, exists := loopContext[parts[0]]
								if exists {
									if mapObj, isMap := obj.(map[string]interface{}); isMap {
										if val, hasAttr := mapObj[parts[1]]; hasAttr {
											sb.WriteString(fmt.Sprintf("%v", val))
											nodeIndex++
											continue
										}
									}
								}
							}
						}
					}

					// Fall back to standard evaluation for other expressions
					val, err := evalExprFunc(node.Content, loopContext)
					if err != nil {
						// Try simple string replacement for complex dot expressions
						trimmed := strings.TrimSpace(node.Content)
						if strings.Contains(trimmed, ".") {
							parts := strings.SplitN(trimmed, ".", 2)
							if len(parts) == 2 && strings.Contains(parts[1], ".") {
								// Nested attribute access - try to resolve step by step
								obj, exists := loopContext[parts[0]]
								if exists {
									if mapObj, isMap := obj.(map[string]interface{}); isMap {
										subParts := strings.Split(parts[1], ".")
										current := mapObj
										for i, part := range subParts {
											if i == len(subParts)-1 {
												if val, exists := current[part]; exists {
													sb.WriteString(fmt.Sprintf("%v", val))
													nodeIndex++
													break
												}
											} else {
												if nextObj, exists := current[part]; exists {
													if nextMap, isMap := nextObj.(map[string]interface{}); isMap {
														current = nextMap
													} else {
														break
													}
												} else {
													break
												}
											}
										}
										continue
									}
								}
							}
						}

						// If all else fails, fall back to evaluateFullExpressionInternal
						val, wasUndef, _ := evaluateFullExpressionInternal(node.Content, loopContext)
						if !wasUndef {
							sb.WriteString(fmt.Sprintf("%v", val))
						}
						nodeIndex++
						continue
					}
					sb.WriteString(fmt.Sprintf("%v", val))
					nodeIndex++

				case NodeControlTag:
					// Properly handle nested control structures
					if node.Control == nil {
						return "", fmt.Errorf("internal parser error: NodeControlTag has nil Control info for content '%s'", node.Content)
					}

					switch node.Control.Type {
					case ControlIf:
						// Delegate to handleIfStatement for nested if blocks
						renderedBlock, nextIdx, err := handleIfStatement(nodes, nodeIndex, loopContext, func(expression string, ctx map[string]interface{}) (interface{}, error) {
							// Special handling for comparison with dot notation variables
							// like "loop.index > 1" or "not loop.last"
							trimmed := strings.TrimSpace(expression)

							// Handle conditions like "not loop.last"
							if strings.HasPrefix(trimmed, "not ") && strings.Contains(trimmed, ".") {
								// Extract variable after "not "
								varName := strings.TrimSpace(strings.TrimPrefix(trimmed, "not "))
								// Check if it's directly a dotted variable with no other operations
								if !strings.Contains(varName, " ") {
									// Evaluate the variable
									val, err := evaluateDotNotation(varName, ctx)
									if err == nil && val != nil {
										// Apply 'not' operator to the result
										return !isTruthy(val), nil
									}
								}
							}

							// Handle expressions like "loop.index > 1"
							if strings.Contains(trimmed, " > ") && strings.Contains(trimmed, ".") {
								parts := strings.Split(trimmed, " > ")
								if len(parts) == 2 {
									leftVar := strings.TrimSpace(parts[0])
									rightVal := strings.TrimSpace(parts[1])

									// If left part contains dot notation, evaluate it first
									if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
										left, err := evaluateDotNotation(leftVar, ctx)
										if err == nil && left != nil {
											// Now create an expression like "2 > 1" that can be evaluated
											newExpr := fmt.Sprintf("%v > %s", left, rightVal)
											result, err := ParseAndEvaluate(newExpr, ctx)
											if err == nil {
												return result, nil
											}
										}
									}
								}
							}

							// Handle expressions like "loop.index <= 1"
							if strings.Contains(trimmed, " <= ") && strings.Contains(trimmed, ".") {
								parts := strings.Split(trimmed, " <= ")
								if len(parts) == 2 {
									leftVar := strings.TrimSpace(parts[0])
									rightVal := strings.TrimSpace(parts[1])

									// If left part contains dot notation, evaluate it first
									if strings.Contains(leftVar, ".") && !strings.Contains(leftVar, " ") {
										left, err := evaluateDotNotation(leftVar, ctx)
										if err == nil && left != nil {
											// Now create an expression like "2 <= 1" that can be evaluated
											newExpr := fmt.Sprintf("%v <= %s", left, rightVal)
											result, err := ParseAndEvaluate(newExpr, ctx)
											if err == nil {
												return result, nil
											}
										}
									}
								}
							}

							// Try LALR parser directly for expressions without complex dot notation
							result, err := ParseAndEvaluate(expression, ctx)
							if err == nil {
								return result, nil
							}

							// For simple expressions, fall back to standard evaluation
							return evalExprFunc(expression, ctx)
						}, customProcessNodesFunc)
						if err != nil {
							return "", err
						}
						sb.WriteString(renderedBlock)
						nodeIndex = nextIdx

					case ControlFor:
						// Delegate to handleForStatement for nested for loops
						renderedBlock, nextIdx, err := handleForStatement(nodes, nodeIndex, loopContext, evalExprFunc, customProcessNodesFunc)
						if err != nil {
							return "", err
						}
						sb.WriteString(renderedBlock)
						nodeIndex = nextIdx

					case ControlEndIf, ControlEndFor, ControlElse, ControlElseIf:
						// These should be handled by their respective handlers
						return "", fmt.Errorf("unexpected control tag '%s' found at node index %d in for loop body. Content: %s",
							node.Control.Type, nodeIndex, node.Content)

					default:
						return "", fmt.Errorf("unhandled control tag type in for loop: %s", node.Control.Type)
					}

				default:
					// Use standard processing for other node types
					result, err := processBlockNodesFunc([](*Node){node}, loopContext)
					if err != nil {
						return "", err
					}
					sb.WriteString(result)
					nodeIndex++
				}
			}

			return sb.String(), nil
		}

		// Process the loop body with the new context and custom processor
		renderedBody, processErr := customProcessNodesFunc(bodyNodes, iterContext)
		if processErr != nil {
			return "", currentIndex, fmt.Errorf("error processing for loop body: %v", processErr)
		}

		result.WriteString(renderedBody)
	}

	return result.String(), endForIndex + 1, nil
}

// evaluateExprInContext evaluates an expression string in a given context
// and returns the result as a string
func evaluateExprInContext(expr string, context map[string]interface{}) (string, error) {
	trimmedExpr := strings.TrimSpace(expr)

	// Try LALR parser first
	result, err := ParseAndEvaluate(trimmedExpr, context)
	if err == nil {
		return fmt.Sprintf("%v", result), nil
	}

	// Fall back to standard evaluation
	result, wasUndef, err := evaluateFullExpressionInternal(trimmedExpr, context)
	if err != nil {
		return "", err
	}

	if wasUndef && result == nil {
		return "", nil
	}

	return fmt.Sprintf("%v", result), nil
}

// convertToSlice converts various types to a slice of interface{} for iteration
func convertToSlice(val interface{}) ([]interface{}, error) {
	if val == nil {
		return []interface{}{}, nil
	}

	switch v := val.(type) {
	case []interface{}:
		return v, nil
	case []string:
		result := make([]interface{}, len(v))
		for i, s := range v {
			result[i] = s
		}
		return result, nil
	case []int:
		result := make([]interface{}, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result, nil
	case map[string]interface{}:
		result := make([]interface{}, 0, len(v))
		for _, val := range v {
			result = append(result, val)
		}
		return result, nil
	case string:
		// Convert string to a sequence of characters
		result := make([]interface{}, len(v))
		for i, c := range v {
			result[i] = string(c)
		}
		return result, nil
	default:
		// Try to use reflection for other slice/array types
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			result := make([]interface{}, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				result[i] = rv.Index(i).Interface()
			}
			return result, nil
		}
		return nil, fmt.Errorf("cannot iterate over type %T", val)
	}
}
