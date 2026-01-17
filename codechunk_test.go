package codechunk

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
)

func TestChunkBasic(t *testing.T) {
	code := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

func helper() int {
	return 42
}
`
	chunks, err := Chunk("main.go", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Verify chunk structure
	for i, chunk := range chunks {
		if chunk.Text == "" {
			t.Errorf("Chunk %d has empty text", i)
		}
		if chunk.Index != i {
			t.Errorf("Chunk %d has incorrect index: %d", i, chunk.Index)
		}
		if chunk.TotalChunks != len(chunks) {
			t.Errorf("Chunk %d has incorrect TotalChunks: %d", i, chunk.TotalChunks)
		}
	}
}

func TestChunkBytes(t *testing.T) {
	code := []byte(`func hello() { return "hi" }`)
	chunks, err := ChunkBytes("test.go", code, nil)
	if err != nil {
		t.Fatalf("ChunkBytes failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkUnsupportedLanguage(t *testing.T) {
	code := `body { color: red; }`
	_, err := Chunk("style.css", code, nil)
	if err != ErrUnsupportedLanguage {
		t.Errorf("Expected ErrUnsupportedLanguage, got: %v", err)
	}
}

func TestChunkWithOptions(t *testing.T) {
	code := `package main

func main() {
	// This is a comment
}
`
	opts := &ChunkOptions{
		MaxChunkSize:  500,
		ContextMode:   ContextModeMinimal,
		SiblingDetail: SiblingDetailNames,
		OverlapLines:  5,
	}

	chunks, err := Chunk("main.go", code, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkContextModeNone(t *testing.T) {
	code := `func hello() { return "hi" }`
	opts := &ChunkOptions{
		ContextMode: ContextModeNone,
	}

	chunks, err := Chunk("test.go", code, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Context should be minimal
	for _, chunk := range chunks {
		if len(chunk.Context.Scope) != 0 {
			t.Error("ContextModeNone should not include scope")
		}
	}
}

func TestChunkTypeScript(t *testing.T) {
	code := `
interface User {
	name: string;
	age: number;
}

function greet(user: User): string {
	return "Hello, " + user.name;
}

class UserService {
	private users: User[] = [];

	addUser(user: User): void {
		this.users.push(user);
	}
}
`
	chunks, err := Chunk("user.ts", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkPython(t *testing.T) {
	code := `
import os
from typing import List

def greet(name: str) -> str:
    """Greet a person."""
    return f"Hello, {name}!"

class Calculator:
    """A simple calculator."""

    def add(self, a: int, b: int) -> int:
        """Add two numbers."""
        return a + b
`
	chunks, err := Chunk("calculator.py", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkRust(t *testing.T) {
	code := `
use std::io;

fn main() {
    println!("Hello, world!");
}

struct Point {
    x: i32,
    y: i32,
}

impl Point {
    fn new(x: i32, y: i32) -> Self {
        Self { x, y }
    }
}
`
	chunks, err := Chunk("main.rs", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkJava(t *testing.T) {
	code := `
package com.example;

import java.util.List;

public class Main {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
    }

    public int add(int a, int b) {
        return a + b;
    }
}
`
	chunks, err := Chunk("Main.java", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkJavaScript(t *testing.T) {
	code := `
import { useState } from 'react';

function Counter() {
    const [count, setCount] = useState(0);
    return <button onClick={() => setCount(count + 1)}>{count}</button>;
}

export default Counter;
`
	chunks, err := Chunk("counter.jsx", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkStream(t *testing.T) {
	code := `package main

func main() {}
func helper() {}
`
	ch, err := ChunkStream("main.go", code, nil)
	if err != nil {
		t.Fatalf("ChunkStream failed: %v", err)
	}

	count := 0
	for chunk := range ch {
		if chunk.TotalChunks != -1 {
			t.Error("Streaming mode should have TotalChunks = -1")
		}
		count++
	}

	if count == 0 {
		t.Error("Expected at least one chunk from stream")
	}
}

func TestChunkStreamUnsupported(t *testing.T) {
	_, err := ChunkStream("file.txt", "hello", nil)
	if err != ErrUnsupportedLanguage {
		t.Errorf("Expected ErrUnsupportedLanguage, got: %v", err)
	}
}

func TestChunkBatch(t *testing.T) {
	files := []FileInput{
		{Filepath: "main.go", Code: `package main; func main() {}`},
		{Filepath: "util.go", Code: `package util; func Helper() int { return 42 }`},
	}

	results := ChunkBatch(files, nil)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if result.Error != nil {
			t.Errorf("Unexpected error for %s: %v", result.Filepath, result.Error)
		}
		if len(result.Chunks) == 0 {
			t.Errorf("Expected chunks for %s", result.Filepath)
		}
	}
}

func TestChunkBatchWithError(t *testing.T) {
	files := []FileInput{
		{Filepath: "main.go", Code: `package main; func main() {}`},
		{Filepath: "style.css", Code: `body { color: red; }`}, // Unsupported
	}

	results := ChunkBatch(files, nil)

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// First should succeed
	if results[0].Error != nil {
		t.Errorf("Expected first file to succeed: %v", results[0].Error)
	}

	// Second should fail
	if results[1].Error == nil {
		t.Error("Expected second file to fail")
	}
}

func TestChunkBatchEmpty(t *testing.T) {
	results := ChunkBatch([]FileInput{}, nil)
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestChunkBatchWithOptions(t *testing.T) {
	files := []FileInput{
		{Filepath: "main.go", Code: `package main; func main() {}`},
	}

	var progressCalls int32
	opts := &BatchOptions{
		ChunkOptions: ChunkOptions{
			MaxChunkSize: 1000,
		},
		Concurrency: 1,
		OnProgress: func(completed, total int, filepath string, success bool) {
			atomic.AddInt32(&progressCalls, 1)
		},
	}

	results := ChunkBatch(files, opts)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if atomic.LoadInt32(&progressCalls) != 1 {
		t.Errorf("Expected 1 progress call, got %d", progressCalls)
	}
}

func TestChunkBatchWithContext(t *testing.T) {
	files := []FileInput{
		{Filepath: "main.go", Code: `package main; func main() {}`},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := ChunkBatchWithContext(ctx, files, nil)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}
}

func TestChunkBatchWithCancelledContext(t *testing.T) {
	files := make([]FileInput, 100)
	for i := range files {
		files[i] = FileInput{
			Filepath: "main.go",
			Code:     `package main; func main() {}`,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	results := ChunkBatchWithContext(ctx, files, &BatchOptions{Concurrency: 1})

	// Some results may be empty due to cancellation
	_ = results
}

func TestChunkBatchStream(t *testing.T) {
	files := []FileInput{
		{Filepath: "main.go", Code: `package main; func main() {}`},
		{Filepath: "util.go", Code: `package util; func Helper() {}`},
	}

	ch := ChunkBatchStream(files, nil)

	count := 0
	for result := range ch {
		if result.Error != nil {
			t.Errorf("Unexpected error: %v", result.Error)
		}
		count++
	}

	if count != 2 {
		t.Errorf("Expected 2 results, got %d", count)
	}
}

func TestChunkBatchStreamEmpty(t *testing.T) {
	ch := ChunkBatchStream([]FileInput{}, nil)

	count := 0
	for range ch {
		count++
	}

	if count != 0 {
		t.Error("Expected 0 results from empty batch")
	}
}

func TestChunkBatchStreamWithContext(t *testing.T) {
	files := []FileInput{
		{Filepath: "main.go", Code: `package main; func main() {}`},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := ChunkBatchStreamWithContext(ctx, files, nil)

	count := 0
	for range ch {
		count++
	}

	if count != 1 {
		t.Errorf("Expected 1 result, got %d", count)
	}
}

func TestFormatChunkWithContext(t *testing.T) {
	text := "func main() {}"
	ctx := ChunkContext{
		Filepath: "src/main.go",
		Language: LanguageGo,
		Scope: []EntityInfo{
			{Name: "main", Type: EntityTypeFunction, Signature: "func main()"},
		},
		Entities: []ChunkEntityInfo{
			{Name: "main", Type: EntityTypeFunction, Signature: "func main()"},
		},
		Siblings: []SiblingInfo{
			{Name: "helper", Type: EntityTypeFunction, Position: "after", Distance: 1},
		},
		Imports: []ImportInfo{
			{Name: "fmt", Source: "fmt"},
		},
	}

	result := FormatChunkWithContext(text, ctx, "")

	// Should contain filepath
	if !strings.Contains(result, "main.go") {
		t.Error("Result should contain filepath")
	}

	// Should contain scope
	if !strings.Contains(result, "Scope") {
		t.Error("Result should contain Scope header")
	}

	// Should contain defines
	if !strings.Contains(result, "Defines") {
		t.Error("Result should contain Defines header")
	}

	// Should contain the original text
	if !strings.Contains(result, text) {
		t.Error("Result should contain original text")
	}
}

func TestFormatChunkWithContextAndOverlap(t *testing.T) {
	text := "func main() {}"
	ctx := ChunkContext{
		Filepath: "main.go",
	}
	overlapText := "// previous chunk content"

	result := FormatChunkWithContext(text, ctx, overlapText)

	// Should contain overlap markers
	if !strings.Contains(result, "# ...") {
		t.Error("Result should contain overlap marker")
	}
	if !strings.Contains(result, overlapText) {
		t.Error("Result should contain overlap text")
	}
}

func TestFormatChunkWithContextEmpty(t *testing.T) {
	text := "func main() {}"
	ctx := ChunkContext{}

	result := FormatChunkWithContext(text, ctx, "")

	// Should just be the text since context is empty
	if result != text {
		t.Errorf("Expected just text, got: %s", result)
	}
}

func TestGetLastPathSegments(t *testing.T) {
	tests := []struct {
		path     string
		n        int
		expected string
	}{
		{"a/b/c/d/e", 3, "c/d/e"},
		{"a/b", 3, "a/b"},
		{"single", 3, "single"},
		{"a/b/c", 3, "a/b/c"},
		{"a/b/c/d", 2, "c/d"},
	}

	for _, tt := range tests {
		result := getLastPathSegments(tt.path, tt.n)
		if result != tt.expected {
			t.Errorf("getLastPathSegments(%q, %d) = %q, want %q", tt.path, tt.n, result, tt.expected)
		}
	}
}

func TestChunker(t *testing.T) {
	chunker := NewChunker(&ChunkOptions{
		MaxChunkSize: 1000,
	})

	code := `package main; func main() {}`
	chunks, err := chunker.Chunk("main.go", code, nil)
	if err != nil {
		t.Fatalf("Chunker.Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkerWithOverride(t *testing.T) {
	chunker := NewChunker(&ChunkOptions{
		MaxChunkSize: 1000,
		ContextMode:  ContextModeMinimal,
	})

	code := `package main; func main() {}`
	opts := &ChunkOptions{
		MaxChunkSize: 500,
		ContextMode:  ContextModeFull,
	}
	chunks, err := chunker.Chunk("main.go", code, opts)
	if err != nil {
		t.Fatalf("Chunker.Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkerNilOptions(t *testing.T) {
	chunker := NewChunker(nil)

	code := `package main; func main() {}`
	chunks, err := chunker.Chunk("main.go", code, nil)
	if err != nil {
		t.Fatalf("Chunker.Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkWithLanguageOverride(t *testing.T) {
	// Use TypeScript content but specify language as TypeScript explicitly
	code := `function hello(): string { return "hi"; }`
	opts := &ChunkOptions{
		Language: LanguageTypeScript,
	}

	chunks, err := Chunk("file.txt", code, opts) // .txt would fail without override
	if err != nil {
		t.Fatalf("Chunk with language override failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkLargeFile(t *testing.T) {
	// Generate a large file with many functions
	var builder strings.Builder
	builder.WriteString("package main\n\n")
	for i := 0; i < 50; i++ {
		builder.WriteString("func function")
		builder.WriteString(string(rune('A' + i%26)))
		builder.WriteString("() {\n")
		builder.WriteString("\t// Some code here\n")
		builder.WriteString("\tx := 1 + 2\n")
		builder.WriteString("\ty := x * 3\n")
		builder.WriteString("\t_ = y\n")
		builder.WriteString("}\n\n")
	}

	code := builder.String()
	opts := &ChunkOptions{
		MaxChunkSize: 500, // Small chunks to force multiple
	}

	chunks, err := Chunk("main.go", code, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks for large file, got %d", len(chunks))
	}

	// Verify all chunks have valid ranges
	for i, chunk := range chunks {
		if chunk.ByteRange.Start < 0 || chunk.ByteRange.End < chunk.ByteRange.Start {
			t.Errorf("Chunk %d has invalid byte range: %v", i, chunk.ByteRange)
		}
		if chunk.LineRange.Start < 0 || chunk.LineRange.End < chunk.LineRange.Start {
			t.Errorf("Chunk %d has invalid line range: %v", i, chunk.LineRange)
		}
	}
}

func TestFileInputWithOptions(t *testing.T) {
	files := []FileInput{
		{
			Filepath: "main.go",
			Code:     `package main; func main() {}`,
			Options: &ChunkOptions{
				MaxChunkSize: 500,
				ContextMode:  ContextModeNone,
			},
		},
	}

	results := ChunkBatch(files, nil)

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error != nil {
		t.Errorf("Unexpected error: %v", results[0].Error)
	}
}
