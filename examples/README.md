# codechunk Examples

This directory contains example programs demonstrating various features of the `codechunk` library.

## Running Examples

From the `codechunk` directory, you can run any example:

```bash
# Run the basic example
go run examples/basic/main.go

# Run the batch processing example
go run examples/batch/main.go

# Run the streaming example
go run examples/streaming/main.go

# Run the options example
go run examples/options/main.go

# Run the context cancellation example
go run examples/context/main.go
```

## Examples Overview

### 1. Basic Usage (`basic/`)

Demonstrates the fundamental usage of the library:
- Chunking source code with default options
- Accessing chunk properties (text, byte range, line range)
- Examining entities in each chunk
- Understanding scope chains and imports
- Viewing raw vs contextualized text

**Key concepts:**
- `codechunk.Chunk()` - Main chunking function
- `CodeChunk` struct - Chunk data structure
- `ChunkContext` - Rich semantic context

### 2. Batch Processing (`batch/`)

Shows how to process multiple files concurrently:
- Creating `FileInput` slices
- Configuring concurrency
- Progress callbacks
- Processing files in different languages simultaneously
- Aggregating results

**Key concepts:**
- `codechunk.ChunkBatch()` - Batch processing
- `BatchOptions` - Batch configuration
- `BatchResult` - Per-file results
- `OnProgress` callback

### 3. Streaming (`streaming/`)

Demonstrates streaming chunks for large files:
- Processing chunks as they are generated
- Memory-efficient processing
- Working with sibling information
- Handling overlap between chunks

**Key concepts:**
- `codechunk.ChunkStream()` - Streaming API
- Channel-based chunk delivery
- Chunk overlap configuration

### 4. Custom Options (`options/`)

Comprehensive demonstration of all configuration options:
- `MaxChunkSize` - Controlling chunk size
- `ContextMode` - Full, minimal, or no context
- `SiblingDetail` - Signatures, names, or none
- `FilterImports` - Only relevant imports
- `OverlapLines` - Chunk overlap
- `Language` - Force language detection
- `Chunker` instance - Reusable configuration

**Key concepts:**
- `ChunkOptions` struct - All options
- Context modes and their effects
- Sibling detail levels
- Contextualized text format

### 5. Context Cancellation (`context/`)

Shows how to use Go's context package:
- Timeout-based cancellation
- Manual cancellation
- Streaming with context
- Graceful shutdown handling

**Key concepts:**
- `codechunk.ChunkBatchWithContext()` - Context-aware batch
- `codechunk.ChunkBatchStreamWithContext()` - Context-aware streaming
- `context.WithTimeout()` and `context.WithCancel()`

## Common Patterns

### Simple Chunking

```go
chunks, err := codechunk.Chunk("file.go", code, nil)
if err != nil {
    log.Fatal(err)
}
for _, chunk := range chunks {
    fmt.Println(chunk.ContextualizedText)
}
```

### Batch Processing with Progress

```go
results := codechunk.ChunkBatch(files, &codechunk.BatchOptions{
    Concurrency: 4,
    OnProgress: func(completed, total int, filepath string, success bool) {
        fmt.Printf("Progress: %d/%d\n", completed, total)
    },
})
```

### Streaming Large Files

```go
ch, err := codechunk.ChunkStream("large.go", code, nil)
if err != nil {
    log.Fatal(err)
}
for chunk := range ch {
    process(chunk)
}
```

### Context Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

results := codechunk.ChunkBatchWithContext(ctx, files, nil)
```

### Reusable Chunker

```go
chunker := codechunk.NewChunker(&codechunk.ChunkOptions{
    MaxChunkSize: 1000,
    ContextMode:  codechunk.ContextModeFull,
})

// Use defaults
chunks1, _ := chunker.Chunk("a.go", codeA, nil)

// Override for specific file
chunks2, _ := chunker.Chunk("b.go", codeB, &codechunk.ChunkOptions{
    MaxChunkSize: 500,
})
```

## Output Format

The `ContextualizedText` field contains formatted output optimized for embedding models:

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

This format provides:
- File path context
- Scope chain (innermost to outermost)
- Entity signatures defined in the chunk
- Import dependencies
- Surrounding code context (siblings)
