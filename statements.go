package jinja

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
				truthy := IsTruthy(conditionResult)

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

	// Check if this is a key-value unpacking pattern (contains a comma)
	keyValueUnpacking := false
	var keyVarName, valueVarName string

	// The expression can be in two formats:
	// 1. "item in items"
	// 2. "key, value in items"
	parts := strings.SplitN(forExpr, " in ", 2)
	if len(parts) != 2 {
		return "", currentIndex, fmt.Errorf("invalid for loop expression: %s", forExpr)
	}

	loopVarOrPair := strings.TrimSpace(parts[0])
	collectionExpr := strings.TrimSpace(parts[1])

	// Check if we have a key-value pair pattern
	if strings.Contains(loopVarOrPair, ",") {
		keyValueUnpacking = true
		pairParts := strings.Split(loopVarOrPair, ",")
		if len(pairParts) != 2 {
			return "", currentIndex, fmt.Errorf("invalid key-value unpacking format in for loop: %s", loopVarOrPair)
		}
		keyVarName = strings.TrimSpace(pairParts[0])
		valueVarName = strings.TrimSpace(pairParts[1])
	} else {
		// If not key-value unpacking, just use the loop variable name directly
		// Don't declare a new variable to avoid unused variable warning
	}

	// Evaluate the collection expression
	var collectionVal interface{}
	var evalErr error

	// First try to evaluate as a compound expression for nested properties
	collectionVal, evalErr = ParseAndEvaluate(collectionExpr, context)
	if evalErr != nil {
		// Fall back to simple evaluation
		collectionVal, evalErr = evalExprFunc(collectionExpr, context)
		if evalErr != nil {
			return "", currentIndex, fmt.Errorf("error evaluating for loop collection '%s': %v", collectionExpr, evalErr)
		}
	}

	// Prepare the results from iterating over the collection
	var result strings.Builder

	if keyValueUnpacking {
		// For key-value unpacking, we need to handle maps differently
		mapVal, ok := collectionVal.(map[string]interface{})
		if !ok {
			// Try to convert to a map
			mapVal, evalErr = convertToMap(collectionVal)
			if evalErr != nil {
				return "", currentIndex, fmt.Errorf("for loop with key-value unpacking requires a dictionary/map collection: %v", evalErr)
			}
		}

		// Create a slice of items for consistent handling of loop.index, etc.
		items := make([]struct {
			Key   interface{}
			Value interface{}
		}, 0, len(mapVal))

		// Convert map entries to key-value pairs
		for k, v := range mapVal {
			items = append(items, struct {
				Key   interface{}
				Value interface{}
			}{k, v})
		}

		// Create loop context for each iteration
		for i, item := range items {
			// Create a copy of the context for this iteration
			iterContext := make(map[string]interface{})
			for k, v := range context {
				iterContext[k] = v
			}

			// Add the key and value variables to the context
			iterContext[keyVarName] = item.Key
			iterContext[valueVarName] = item.Value

			// Add the 'loop' special variable with proper primitive types for index values
			// to ensure they're evaluated correctly in templates
			loopInfo := map[string]interface{}{
				"index":     i + 1,              // 1-based index
				"index0":    i,                  // 0-based index
				"first":     i == 0,             // True if first iteration
				"last":      i == len(items)-1,  // True if last iteration
				"length":    len(items),         // Total number of items
				"revindex":  len(items) - i,     // Reverse index (1-based)
				"revindex0": len(items) - i - 1, // Reverse index (0-based)
			}
			iterContext["loop"] = loopInfo

			// Process the body of the loop with this context
			renderedNodes, err := processBlockNodesFunc(bodyNodes, iterContext)
			if err != nil {
				return "", currentIndex, fmt.Errorf("error processing for loop body: %v", err)
			}
			result.WriteString(renderedNodes)
		}
	} else {
		// Original behavior for simple item iteration
		loopVarName := strings.TrimSpace(loopVarOrPair)

		// Convert the collection to a slice for iteration
		collection, err := convertToSlice(collectionVal)
		if err != nil {
			return "", currentIndex, fmt.Errorf("for loop requires an iterable collection: %v", err)
		}

		// Create loop context for each iteration
		for i, item := range collection {
			// Create a copy of the context for this iteration
			iterContext := make(map[string]interface{})
			for k, v := range context {
				iterContext[k] = v
			}

			// Add the loop variable to the context
			iterContext[loopVarName] = item

			// Add the 'loop' special variable with primitive types for index values
			// to ensure they're evaluated correctly in templates
			loopInfo := map[string]interface{}{
				"index":     i + 1,                   // 1-based index as int
				"index0":    i,                       // 0-based index as int
				"first":     i == 0,                  // True if first iteration
				"last":      i == len(collection)-1,  // True if last iteration
				"length":    len(collection),         // Total number of items as int
				"revindex":  len(collection) - i,     // Reverse index (1-based) as int
				"revindex0": len(collection) - i - 1, // Reverse index (0-based) as int
			}
			iterContext["loop"] = loopInfo

			// Process the body of the loop with this context
			renderedNodes, err := processBlockNodesFunc(bodyNodes, iterContext)
			if err != nil {
				return "", currentIndex, fmt.Errorf("error processing for loop body: %v", err)
			}
			result.WriteString(renderedNodes)
		}
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

// convertToMap converts a value to a map[string]interface{} if possible.
// This is used for key-value unpacking in for loops.
func convertToMap(val interface{}) (map[string]interface{}, error) {
	if val == nil {
		return map[string]interface{}{}, nil
	}

	// If it's already a map[string]interface{}, return it
	if m, ok := val.(map[string]interface{}); ok {
		return m, nil
	}

	// If it's a map with different key types, convert it
	if reflect.TypeOf(val).Kind() == reflect.Map {
		mapVal := reflect.ValueOf(val)
		result := make(map[string]interface{}, mapVal.Len())

		// Iterate through the map entries
		iter := mapVal.MapRange()
		for iter.Next() {
			// Convert key to string
			key := fmt.Sprintf("%v", iter.Key().Interface())
			// Get the value
			value := iter.Value().Interface()
			result[key] = value
		}

		return result, nil
	}

	// For other types, check if they're map-like
	// For example, a struct could be converted to a map of field names to values
	if reflect.TypeOf(val).Kind() == reflect.Struct {
		structVal := reflect.ValueOf(val)
		structType := structVal.Type()
		result := make(map[string]interface{}, structType.NumField())

		// Iterate through the struct fields
		for i := 0; i < structType.NumField(); i++ {
			fieldName := structType.Field(i).Name
			fieldValue := structVal.Field(i).Interface()
			result[fieldName] = fieldValue
		}

		return result, nil
	}

	return nil, fmt.Errorf("cannot convert %T to a map", val)
}
