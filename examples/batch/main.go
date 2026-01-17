// Example: Batch Processing
//
// This example demonstrates how to process multiple files concurrently
// using the batch processing API.
package main

import (
	"fmt"
	"sync/atomic"
	"time"

	codechunk "github.com/pc-coder/go-code-chunk"
)

func main() {
	// Sample files to process
	files := []codechunk.FileInput{
		{
			Filepath: "src/user.go",
			Code: `package user

type User struct {
	ID   string
	Name string
}

func NewUser(name string) *User {
	return &User{Name: name}
}

func (u *User) GetName() string {
	return u.Name
}
`,
		},
		{
			Filepath: "src/product.ts",
			Code: `interface Product {
	id: string;
	name: string;
	price: number;
}

class ProductService {
	private products: Product[] = [];

	addProduct(product: Product): void {
		this.products.push(product);
	}

	getProduct(id: string): Product | undefined {
		return this.products.find(p => p.id === id);
	}

	listProducts(): Product[] {
		return [...this.products];
	}
}

export { Product, ProductService };
`,
		},
		{
			Filepath: "src/utils.py",
			Code: `"""Utility functions for the application."""

from typing import List, Optional

def filter_none(items: List[Optional[str]]) -> List[str]:
    """Filter out None values from a list.

    Args:
        items: List that may contain None values.

    Returns:
        List with None values removed.
    """
    return [item for item in items if item is not None]

class StringUtils:
    """Utility class for string operations."""

    @staticmethod
    def capitalize_words(text: str) -> str:
        """Capitalize the first letter of each word."""
        return ' '.join(word.capitalize() for word in text.split())

    @staticmethod
    def reverse(text: str) -> str:
        """Reverse a string."""
        return text[::-1]
`,
		},
		{
			Filepath: "src/handler.rs",
			Code: `use std::collections::HashMap;

/// A simple key-value store handler.
pub struct Handler {
    store: HashMap<String, String>,
}

impl Handler {
    /// Creates a new Handler instance.
    pub fn new() -> Self {
        Handler {
            store: HashMap::new(),
        }
    }

    /// Sets a value for the given key.
    pub fn set(&mut self, key: String, value: String) {
        self.store.insert(key, value);
    }

    /// Gets the value for the given key.
    pub fn get(&self, key: &str) -> Option<&String> {
        self.store.get(key)
    }

    /// Removes a key from the store.
    pub fn delete(&mut self, key: &str) -> Option<String> {
        self.store.remove(key)
    }
}
`,
		},
		{
			Filepath: "src/Main.java",
			Code: `package com.example;

import java.util.ArrayList;
import java.util.List;

/**
 * Main application class.
 */
public class Main {
    private List<String> items;

    public Main() {
        this.items = new ArrayList<>();
    }

    /**
     * Adds an item to the list.
     * @param item The item to add.
     */
    public void addItem(String item) {
        items.add(item);
    }

    /**
     * Gets all items.
     * @return List of items.
     */
    public List<String> getItems() {
        return new ArrayList<>(items);
    }

    public static void main(String[] args) {
        Main app = new Main();
        app.addItem("Hello");
        app.addItem("World");
        System.out.println(app.getItems());
    }
}
`,
		},
	}

	fmt.Printf("Processing %d files...\n\n", len(files))

	// Track progress
	var processedCount int32

	// Configure batch options
	opts := &codechunk.BatchOptions{
		Concurrency: 4, // Process up to 4 files concurrently
		ChunkOptions: codechunk.ChunkOptions{
			MaxChunkSize:  1000,
			ContextMode:   codechunk.ContextModeFull,
			SiblingDetail: codechunk.SiblingDetailSignatures,
		},
		OnProgress: func(completed, total int, filepath string, success bool) {
			atomic.AddInt32(&processedCount, 1)
			status := "OK"
			if !success {
				status = "FAILED"
			}
			fmt.Printf("[%d/%d] %s: %s\n", completed, total, filepath, status)
		},
	}

	// Process files in batch
	start := time.Now()
	results := codechunk.ChunkBatch(files, opts)
	elapsed := time.Since(start)

	fmt.Printf("\nProcessed %d files in %v\n\n", len(results), elapsed)

	// Print results summary
	totalChunks := 0
	totalEntities := 0

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Error processing %s: %v\n", result.Filepath, result.Error)
			continue
		}

		chunkCount := len(result.Chunks)
		entityCount := 0
		for _, chunk := range result.Chunks {
			entityCount += len(chunk.Context.Entities)
		}

		totalChunks += chunkCount
		totalEntities += entityCount

		fmt.Printf("%-20s: %d chunks, %d entities\n",
			result.Filepath, chunkCount, entityCount)

		// Print entity details
		for _, chunk := range result.Chunks {
			for _, entity := range chunk.Context.Entities {
				if entity.Type != codechunk.EntityTypeImport {
					fmt.Printf("  - %s (%s)\n", entity.Name, entity.Type)
				}
			}
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Total files:    %d\n", len(results))
	fmt.Printf("Total chunks:   %d\n", totalChunks)
	fmt.Printf("Total entities: %d\n", totalEntities)
}
