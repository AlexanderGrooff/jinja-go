# Performance Optimization Recommendations

Based on profiling results, here are the key findings and recommendations for improving the performance of the Jinja template engine implementation.

## Key Findings

1. **Memory Allocations**: The majority of time is spent in memory allocation and garbage collection
   - `handleForStatement` accounts for ~80% of allocations in nested loops
   - String operations like `WriteString` and `genSplit` are major allocation sources
   - Parser and expression evaluator create many short-lived objects

2. **Parsing Overhead**: Each template is parsed from scratch for every rendering
   - `parseAttributeAccess`, `parseExpression`, and `ParseNext` are hotspots
   - No caching of parsed templates between renderings

3. **String Handling**: Excessive string operations
   - `splitExpressionWithFilters` and `tokenizeIdentifierOrKeyword` show up prominently
   - Many string allocations in parser and lexer

4. **Control Structure Handling**: Significant overhead
   - `handleForStatement` and `parseControlTagDetail` are expensive
   - Complex nested structures (loops within loops) have multiplicative overhead

## Recommendations

### 1. Implement Template Caching

**Priority: High**

Create a caching layer to store parsed templates:

```go
type TemplateCache struct {
    cache map[string][]*Node
    mu    sync.RWMutex
}

func (tc *TemplateCache) Get(template string) ([]*Node, bool) {
    tc.mu.RLock()
    defer tc.mu.RUnlock()
    nodes, ok := tc.cache[template]
    return nodes, ok
}

func (tc *TemplateCache) Set(template string, nodes []*Node) {
    tc.mu.Lock()
    defer tc.mu.Unlock()
    tc.cache[template] = nodes
}
```

Modify `TemplateString` to use the cache:

```go
var templateCache = &TemplateCache{cache: make(map[string][]*Node)}

func TemplateString(template string, context map[string]interface{}) (string, error) {
    nodes, found := templateCache.Get(template)
    if !found {
        parser := NewParser(template)
        var err error
        nodes, err = parser.ParseAll()
        if err != nil {
            return "", err
        }
        templateCache.Set(template, nodes)
    }
    
    return renderNodes(nodes, context)
}
```

### 2. Reduce Allocations in Parser

**Priority: High**

1. **Pre-allocate slices**: For token lists, node lists, etc.

```go
// Before
var nodes []*Node

// After
nodes := make([]*Node, 0, 10) // Estimate initial capacity
```

2. **Use string pools and object pools**:

```go
var nodePool = sync.Pool{
    New: func() interface{} {
        return &Node{}
    },
}

// Get a node from the pool
node := nodePool.Get().(*Node)
// Reset its fields
*node = Node{Type: NodeText, Content: ""}
// Use the node...
// Return to pool when done
nodePool.Put(node)
```

3. **Avoid unnecessary string copying**: 
   - Use string indexing rather than slicing where possible
   - Use `strings.Builder` with capacity hints

### 3. Optimize String Operations

**Priority: Medium**

1. **Rewrite `splitExpressionWithFilters`** to reduce allocations:
   - Current implementation creates many intermediate strings
   - Consider a single-pass approach that records start/end positions

2. **Optimize `parseControlTagDetail`**:
   - Avoid `strings.Fields` which allocates a new slice
   - Consider a state machine approach that doesn't split the string

3. **Use byte slices instead of strings** for internal parsing operations:
   - Many string operations can be replaced with byte operations
   - This reduces allocations and copying

### 4. Implement a Tokenizer Stage

**Priority: Medium**

Add a separate tokenization stage that happens once per template:

```go
type TokenType int

const (
    TokenText TokenType = iota
    TokenExpressionStart
    TokenExpressionEnd
    // etc.
)

type Token struct {
    Type    TokenType
    Content string
    Pos     int
}

func Tokenize(template string) ([]Token, error) {
    // Implementation
}
```

Then make the parser work on tokens instead of raw strings.

### 5. Optimize Loop Handling

**Priority: High**

1. **Refactor `handleForStatement`**: 
   - This is the biggest allocation source
   - Cache loop variable lookups
   - Avoid repeatedly building the `loop` object

2. **Use more efficient loop iteration**:
   - Pre-compute loop length once
   - Reuse loop context objects when possible

### 6. Consider Compilation to Functions

**Priority: Low (Future Enhancement)**

For a more radical improvement, consider compiling templates to Go functions:

```go
type CompiledTemplate func(context map[string]interface{}) (string, error)

func Compile(template string) (CompiledTemplate, error) {
    // Parse template into an AST
    // Generate a function that directly implements the template logic
    // Return the function
}
```

This eliminates most runtime parsing and evaluation overhead.

## Implementation Plan

1. First implement template caching (biggest immediate win)
2. Then focus on reducing allocations in the parser
3. Next optimize string operations
4. Finally implement more complex changes like tokenization

The improvements should be benchmarked at each stage to ensure they're effective. 