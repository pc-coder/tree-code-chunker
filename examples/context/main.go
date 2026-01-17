// Example: Context Cancellation
//
// This example demonstrates how to use Go's context package
// for cancellation and timeouts with batch processing.
package main

import (
	"context"
	"fmt"
	"time"

	codechunk "github.com/pc-coder/go-code-chunk"
)

func main() {
	// Generate many files to process
	files := generateFiles(100)

	fmt.Printf("Processing %d files...\n\n", len(files))

	// Example 1: Cancellation with timeout
	fmt.Println("=== Example 1: With Timeout ===")
	demonstrateTimeout(files)

	// Example 2: Manual cancellation
	fmt.Println("\n=== Example 2: Manual Cancellation ===")
	demonstrateManualCancellation(files)

	// Example 3: Streaming with context
	fmt.Println("\n=== Example 3: Streaming with Context ===")
	demonstrateStreamingWithContext(files)
}

func demonstrateTimeout(files []codechunk.FileInput) {
	// Create a context with a 100ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()

	// Process with context - will be cancelled after timeout
	results := codechunk.ChunkBatchWithContext(ctx, files, &codechunk.BatchOptions{
		Concurrency: 2, // Slow processing to demonstrate timeout
	})

	elapsed := time.Since(start)

	// Count completed vs cancelled
	completed := 0
	for _, r := range results {
		if r.Error == nil && r.Chunks != nil {
			completed++
		}
	}

	fmt.Printf("Completed %d/%d files in %v (timeout was 100ms)\n",
		completed, len(files), elapsed)

	// Check if context was cancelled
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Println("Context deadline exceeded - processing was cancelled")
	}
}

func demonstrateManualCancellation(files []codechunk.FileInput) {
	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Counter for processed files
	processedCh := make(chan int, len(files))

	// Start processing in a goroutine
	resultsCh := make(chan []codechunk.BatchResult, 1)
	go func() {
		results := codechunk.ChunkBatchWithContext(ctx, files, &codechunk.BatchOptions{
			Concurrency: 4,
			OnProgress: func(completed, total int, filepath string, success bool) {
				processedCh <- completed
			},
		})
		resultsCh <- results
	}()

	// Cancel after processing 10 files
	cancelAfter := 10
	processed := 0

	for {
		select {
		case p := <-processedCh:
			processed = p
			if processed >= cancelAfter {
				fmt.Printf("Processed %d files, cancelling...\n", processed)
				cancel()
			}
		case results := <-resultsCh:
			// Processing finished (either completed or cancelled)
			completed := 0
			for _, r := range results {
				if r.Error == nil && r.Chunks != nil {
					completed++
				}
			}
			fmt.Printf("Final result: %d/%d files completed\n", completed, len(files))
			return
		}
	}
}

func demonstrateStreamingWithContext(files []codechunk.FileInput) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()

	// Stream results with context
	ch := codechunk.ChunkBatchStreamWithContext(ctx, files[:20], &codechunk.BatchOptions{
		Concurrency: 2,
	})

	// Process results as they arrive
	completed := 0
	cancelled := false

	for result := range ch {
		if result.Error != nil {
			// Check if error is due to cancellation
			if ctx.Err() != nil {
				cancelled = true
				break
			}
			fmt.Printf("Error processing %s: %v\n", result.Filepath, result.Error)
			continue
		}

		completed++
		fmt.Printf("Received result for %s: %d chunks\n",
			result.Filepath, len(result.Chunks))
	}

	elapsed := time.Since(start)

	fmt.Printf("\nCompleted: %d files in %v\n", completed, elapsed)
	if cancelled {
		fmt.Println("Processing was cancelled due to timeout")
	}
}

// generateFiles creates sample files for testing
func generateFiles(count int) []codechunk.FileInput {
	files := make([]codechunk.FileInput, count)

	templates := []struct {
		ext      string
		template string
	}{
		{".go", `package file%d

import "fmt"

type Service%d struct {
	name string
}

func NewService%d() *Service%d {
	return &Service%d{name: "service%d"}
}

func (s *Service%d) Run() {
	fmt.Println("Running", s.name)
}

func helper%d() string {
	return "helper result"
}
`},
		{".ts", `interface Config%d {
	name: string;
	value: number;
}

class Handler%d {
	private config: Config%d;

	constructor(config: Config%d) {
		this.config = config;
	}

	process(): string {
		return this.config.name;
	}
}

export function create%d(): Handler%d {
	return new Handler%d({ name: "handler%d", value: %d });
}
`},
		{".py", `"""Module %d - Sample Python file."""

from typing import List, Optional

class Processor%d:
    """Processor class for file %d."""

    def __init__(self, name: str):
        self.name = name
        self.data: List[str] = []

    def process(self, item: str) -> Optional[str]:
        """Process an item."""
        if item:
            self.data.append(item)
            return f"Processed: {item}"
        return None

def create_processor%d() -> Processor%d:
    """Create a new processor."""
    return Processor%d("processor%d")
`},
		{".rs", `//! Module %d

use std::collections::HashMap;

/// Config for module %d
pub struct Config%d {
    pub name: String,
    pub settings: HashMap<String, String>,
}

impl Config%d {
    /// Create a new config
    pub fn new(name: &str) -> Self {
        Config%d {
            name: name.to_string(),
            settings: HashMap::new(),
        }
    }

    /// Get a setting value
    pub fn get(&self, key: &str) -> Option<&String> {
        self.settings.get(key)
    }
}
`},
	}

	for i := 0; i < count; i++ {
		tmpl := templates[i%len(templates)]
		files[i] = codechunk.FileInput{
			Filepath: fmt.Sprintf("src/file%d%s", i, tmpl.ext),
			Code:     fmt.Sprintf(tmpl.template, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i, i),
		}
	}

	return files
}
