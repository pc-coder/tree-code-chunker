# codechunk

AST-aware code chunking for semantic search and RAG pipelines in Go.

[![Go Reference](https://pkg.go.dev/badge/github.com/pc-coder/tree-code-chunker.svg)](https://pkg.go.dev/github.com/pc-coder/tree-code-chunker)
[![Test Coverage](https://img.shields.io/badge/coverage-90.2%25-brightgreen.svg)](https://github.com/pc-coder/tree-code-chunker)

## Overview

`codechunk` uses [tree-sitter](https://tree-sitter.github.io/tree-sitter/) to split source code at semantic boundaries (functions, classes, methods) rather than arbitrary character limits. Each chunk includes rich context: scope chain, imports, siblings, and entity signatures.

This is a Go port of the TypeScript [code-chunk](https://github.com/supermemoryai/code-chunk) library.

## Features

- **AST-aware chunking**: Splits at semantic boundaries, never mid-function
- **Rich context**: Scope chain, imports, siblings, entity signatures
- **Contextualized text**: Pre-formatted for embedding models
- **Multi-language support**: Go, TypeScript, JavaScript, Python, Rust, Java
- **Batch processing**: Process entire codebases with controlled concurrency
- **Streaming API**: Process large files incrementally
- **Context cancellation**: Full support for Go's context package

## Installation

```bash
go get github.com/pc-coder/tree-code-chunker
```

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    "github.com/pc-coder/tree-code-chunker"
)

func main() {
    code := `
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}

func helper() string {
    return "I help!"
}
`

    chunks, err := codechunk.Chunk("main.go", code, nil)
    if err != nil {
        log.Fatal(err)
    }

    for i, chunk := range chunks {
        fmt.Printf("Chunk %d/%d:\n", chunk.Index+1, chunk.TotalChunks)
        fmt.Printf("  Lines: %d-%d\n", chunk.LineRange.Start, chunk.LineRange.End)
        fmt.Printf("  Entities: %d\n", len(chunk.Context.Entities))
        fmt.Println()
    }
}
```

## API Reference

### Core Functions

#### `Chunk(filepath, code string, opts *ChunkOptions) ([]CodeChunk, error)`

Chunks source code into pieces with semantic context.

```go
chunks, err := codechunk.Chunk("src/user.go", sourceCode, &codechunk.ChunkOptions{
    MaxChunkSize:  1500,
    ContextMode:   codechunk.ContextModeFull,
    SiblingDetail: codechunk.SiblingDetailSignatures,
    OverlapLines:  10,
})
```

#### `ChunkBytes(filepath string, code []byte, opts *ChunkOptions) ([]CodeChunk, error)`

Same as `Chunk` but accepts `[]byte` instead of `string`.

#### `ChunkStream(filepath, code string, opts *ChunkOptions) (<-chan CodeChunk, error)`

Streams chunks as they are generated. Useful for large files.

```go
ch, err := codechunk.ChunkStream("large.go", code, nil)
if err != nil {
    log.Fatal(err)
}

for chunk := range ch {
    // Process each chunk as it's generated
    fmt.Println(chunk.Text)
}
```

#### `ChunkBatch(files []FileInput, opts *BatchOptions) []BatchResult`

Processes multiple files concurrently.

```go
files := []codechunk.FileInput{
    {Filepath: "a.go", Code: aCode},
    {Filepath: "b.go", Code: bCode},
    {Filepath: "c.go", Code: cCode},
}

results := codechunk.ChunkBatch(files, &codechunk.BatchOptions{
    Concurrency: 4,
    OnProgress: func(completed, total int, filepath string, success bool) {
        fmt.Printf("Progress: %d/%d\n", completed, total)
    },
})
```

#### `ChunkBatchWithContext(ctx context.Context, files []FileInput, opts *BatchOptions) []BatchResult`

Same as `ChunkBatch` with context support for cancellation.

#### `ChunkBatchStream(files []FileInput, opts *BatchOptions) <-chan BatchResult`

Streams batch results as files complete processing.

#### `ChunkBatchStreamWithContext(ctx context.Context, files []FileInput, opts *BatchOptions) <-chan BatchResult`

Same as `ChunkBatchStream` with context support.

### Types

#### `ChunkOptions`

```go
type ChunkOptions struct {
    MaxChunkSize  int           // Target chunk size in NWS characters (default: 1500)
    ContextMode   ContextMode   // How much context to include (default: ContextModeFull)
    SiblingDetail SiblingDetail // Detail level for siblings (default: SiblingDetailSignatures)
    Language      Language      // Force language (auto-detected if empty)
    OverlapLines  int           // Lines of overlap between chunks (default: 10)
    FilterImports bool          // Only include relevant imports
}
```

#### `CodeChunk`

```go
type CodeChunk struct {
    Text               string       // Raw chunk text
    ContextualizedText string       // Text with context prepended
    ByteRange          ByteRange    // Byte offsets in source
    LineRange          LineRange    // Line numbers in source
    Context            ChunkContext // Rich semantic context
    Index              int          // Chunk index (0-based)
    TotalChunks        int          // Total number of chunks
}
```

#### `ChunkContext`

```go
type ChunkContext struct {
    Filepath   string            // Source file path
    Language   Language          // Detected language
    Scope      []EntityInfo      // Scope chain (innermost first)
    Entities   []ChunkEntityInfo // Entities in this chunk
    Siblings   []SiblingInfo     // Nearby entities
    Imports    []ImportInfo      // Relevant imports
    ParseError error             // Parse error if any
}
```

### Constants

#### Context Modes

```go
const (
    ContextModeFull    ContextMode = "full"    // Include all context
    ContextModeMinimal ContextMode = "minimal" // Minimal context
    ContextModeNone    ContextMode = "none"    // No context
)
```

#### Sibling Detail Levels

```go
const (
    SiblingDetailSignatures SiblingDetail = "signatures" // Include signatures
    SiblingDetailNames      SiblingDetail = "names"      // Only names
    SiblingDetailNone       SiblingDetail = "none"       // No siblings
)
```

#### Supported Languages

```go
const (
    LanguageGo         Language = "go"
    LanguageTypeScript Language = "typescript"
    LanguageJavaScript Language = "javascript"
    LanguagePython     Language = "python"
    LanguageRust       Language = "rust"
    LanguageJava       Language = "java"
)
```

### Utility Functions

#### `DetectLanguage(filepath string) Language`

Detects language from file extension.

```go
lang := codechunk.DetectLanguage("src/main.rs") // Returns LanguageRust
```

#### `FormatChunkWithContext(text string, ctx ChunkContext, overlapText string) string`

Formats chunk text with semantic context prepended.

#### `IsDocComment(text string, lang Language) bool`

Checks if a comment is a documentation comment.

#### `ClearGrammarCache()`

Clears the cached tree-sitter grammars.

### Chunker Instance

For reusing options across multiple calls:

```go
chunker := codechunk.NewChunker(&codechunk.ChunkOptions{
    MaxChunkSize: 1000,
    ContextMode:  codechunk.ContextModeFull,
})

// Use default options
chunks1, _ := chunker.Chunk("a.go", codeA, nil)

// Override specific options
chunks2, _ := chunker.Chunk("b.go", codeB, &codechunk.ChunkOptions{
    MaxChunkSize: 500,
})
```

## Contextualized Output Format

When using `ContextModeFull`, the `ContextualizedText` field contains formatted context:

```
# src/services/user.go
# Scope: UserService > GetUser
# Defines: func GetUser(id string) (*User, error)
# Uses: fmt, errors, database
# After: CreateUser
# Before: DeleteUser

func GetUser(id string) (*User, error) {
    // ... actual code ...
}
```

This format is optimized for embedding models and semantic search.

## How It Works

1. **Parse**: Uses tree-sitter to parse source code into an AST
2. **Extract Entities**: Identifies functions, classes, methods, types, imports
3. **Build Scope Tree**: Creates a hierarchical scope structure
4. **Chunk**: Uses a greedy algorithm to assign AST nodes to chunks based on NWS (non-whitespace) character count
5. **Context**: Enriches each chunk with scope chain, imports, and sibling information
6. **Format**: Generates contextualized text for embedding

### NWS Character Counting

Chunk sizes are measured in non-whitespace characters, which provides a more consistent measure across different coding styles and indentation preferences.

### Greedy Window Assignment

The chunking algorithm:
1. Processes AST nodes in order
2. Adds nodes to current chunk while under `MaxChunkSize`
3. When a node would exceed the limit, starts a new chunk
4. Oversized nodes are split at children or line boundaries
5. Adjacent windows are merged when possible

## Examples

See the [examples](./examples/) directory for complete examples:

- [Basic Usage](./examples/basic/main.go) - Simple chunking
- [Batch Processing](./examples/batch/main.go) - Processing multiple files
- [Streaming](./examples/streaming/main.go) - Streaming chunks
- [Custom Options](./examples/options/main.go) - Configuring chunk options
- [Context Cancellation](./examples/context/main.go) - Using context for cancellation

## Performance

- **O(1) NWS queries**: Uses cumulative sum preprocessing
- **Parallel batch processing**: Configurable concurrency
- **Grammar caching**: Tree-sitter grammars are cached
- **Streaming support**: Memory-efficient processing of large files

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - see [LICENSE](./LICENSE) for details.

## Acknowledgments

- Original TypeScript implementation: [supermemoryai/code-chunk](https://github.com/supermemoryai/code-chunk)
- Tree-sitter Go bindings: [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter)
