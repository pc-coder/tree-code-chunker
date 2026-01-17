// Example: Streaming
//
// This example demonstrates how to use the streaming API to process
// chunks as they are generated, which is useful for large files.
package main

import (
	"fmt"
	"log"
	"strings"

	codechunk "github.com/pc-coder/tree-code-chunker"
)

func main() {
	// Generate a large code file with many functions
	code := generateLargeCode()

	fmt.Printf("Processing large file (%d bytes, %d lines)...\n\n",
		len(code), strings.Count(code, "\n")+1)

	// Stream chunks as they are generated
	ch, err := codechunk.ChunkStream("large.go", code, &codechunk.ChunkOptions{
		MaxChunkSize:  500, // Small chunks to generate more
		ContextMode:   codechunk.ContextModeFull,
		SiblingDetail: codechunk.SiblingDetailNames,
		OverlapLines:  5,
	})
	if err != nil {
		log.Fatalf("Failed to start streaming: %v", err)
	}

	// Process chunks as they arrive
	totalBytes := 0
	for chunk := range ch {
		totalBytes += len(chunk.Text)

		fmt.Printf("Received chunk %d (lines %d-%d, %d bytes)\n",
			chunk.Index+1,
			chunk.LineRange.Start+1,
			chunk.LineRange.End+1,
			len(chunk.Text))

		// Print entities in this chunk
		for _, entity := range chunk.Context.Entities {
			if entity.Type != codechunk.EntityTypeImport {
				fmt.Printf("  - %s (%s)\n", entity.Name, entity.Type)
			}
		}

		// Print siblings
		if len(chunk.Context.Siblings) > 0 {
			beforeSiblings := []string{}
			afterSiblings := []string{}
			for _, s := range chunk.Context.Siblings {
				if s.Position == "before" {
					beforeSiblings = append(beforeSiblings, s.Name)
				} else {
					afterSiblings = append(afterSiblings, s.Name)
				}
			}
			if len(beforeSiblings) > 0 {
				fmt.Printf("  After: %s\n", strings.Join(beforeSiblings, ", "))
			}
			if len(afterSiblings) > 0 {
				fmt.Printf("  Before: %s\n", strings.Join(afterSiblings, ", "))
			}
		}
	}

	fmt.Printf("\n--- Summary ---\n")
	fmt.Printf("Total bytes processed: %d\n", totalBytes)
}

// generateLargeCode creates a sample Go file with many functions
func generateLargeCode() string {
	var sb strings.Builder

	sb.WriteString(`package main

import (
	"fmt"
	"strings"
	"strconv"
)

// Config holds application configuration.
type Config struct {
	Name    string
	Debug   bool
	MaxSize int
}

// App is the main application struct.
type App struct {
	config Config
	data   map[string]interface{}
}

// NewApp creates a new App instance.
func NewApp(config Config) *App {
	return &App{
		config: config,
		data:   make(map[string]interface{}),
	}
}

`)

	// Generate multiple helper functions
	for i := 1; i <= 20; i++ {
		sb.WriteString(fmt.Sprintf(`// Helper%d performs helper operation %d.
func Helper%d(input string) string {
	// Process the input
	result := strings.ToUpper(input)
	result = strings.TrimSpace(result)

	// Add some transformation
	if len(result) > 10 {
		result = result[:10] + "..."
	}

	return fmt.Sprintf("Helper%d: %%s", result)
}

`, i, i, i, i))
	}

	// Add some methods
	sb.WriteString(`// Start initializes and starts the application.
func (a *App) Start() error {
	fmt.Println("Starting application:", a.config.Name)
	return nil
}

// Stop gracefully stops the application.
func (a *App) Stop() error {
	fmt.Println("Stopping application")
	return nil
}

// Set stores a value in the app's data store.
func (a *App) Set(key string, value interface{}) {
	a.data[key] = value
}

// Get retrieves a value from the app's data store.
func (a *App) Get(key string) (interface{}, bool) {
	val, ok := a.data[key]
	return val, ok
}

// ProcessBatch processes multiple items in batch.
func (a *App) ProcessBatch(items []string) []string {
	results := make([]string, len(items))
	for i, item := range items {
		results[i] = a.process(item)
	}
	return results
}

func (a *App) process(item string) string {
	return strings.ToLower(item)
}

func main() {
	config := Config{
		Name:    "MyApp",
		Debug:   true,
		MaxSize: 100,
	}

	app := NewApp(config)
	app.Start()

	// Use helpers
	fmt.Println(Helper1("test"))
	fmt.Println(Helper5("hello"))

	app.Stop()
}
`)

	return sb.String()
}
