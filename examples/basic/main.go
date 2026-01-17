// Example: Basic Usage
//
// This example demonstrates the basic usage of the codechunk library
// to split source code into semantic chunks.
package main

import (
	"fmt"
	"log"

	codechunk "github.com/pc-coder/go-code-chunk"
)

func main() {
	// Sample Go code to chunk
	code := `package main

import (
	"fmt"
	"strings"
)

// User represents a user in the system.
type User struct {
	ID    string
	Name  string
	Email string
}

// NewUser creates a new user with the given name and email.
func NewUser(name, email string) *User {
	return &User{
		ID:    generateID(),
		Name:  name,
		Email: email,
	}
}

// Greet returns a greeting message for the user.
func (u *User) Greet() string {
	return fmt.Sprintf("Hello, %s!", u.Name)
}

// UpdateEmail updates the user's email address.
func (u *User) UpdateEmail(email string) error {
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email: %s", email)
	}
	u.Email = email
	return nil
}

func generateID() string {
	return "user-123"
}

func main() {
	user := NewUser("Alice", "alice@example.com")
	fmt.Println(user.Greet())
}
`

	// Chunk the code with default options
	chunks, err := codechunk.Chunk("user.go", code, nil)
	if err != nil {
		log.Fatalf("Failed to chunk code: %v", err)
	}

	fmt.Printf("Generated %d chunks\n\n", len(chunks))

	// Print information about each chunk
	for _, chunk := range chunks {
		fmt.Printf("=== Chunk %d/%d ===\n", chunk.Index+1, chunk.TotalChunks)
		fmt.Printf("Lines: %d-%d\n", chunk.LineRange.Start+1, chunk.LineRange.End+1)
		fmt.Printf("Bytes: %d-%d\n", chunk.ByteRange.Start, chunk.ByteRange.End)

		// Print entities in this chunk
		if len(chunk.Context.Entities) > 0 {
			fmt.Println("Entities:")
			for _, entity := range chunk.Context.Entities {
				fmt.Printf("  - %s (%s)\n", entity.Name, entity.Type)
				if entity.Signature != "" {
					fmt.Printf("    Signature: %s\n", entity.Signature)
				}
			}
		}

		// Print scope chain
		if len(chunk.Context.Scope) > 0 {
			fmt.Print("Scope: ")
			for i, scope := range chunk.Context.Scope {
				if i > 0 {
					fmt.Print(" > ")
				}
				fmt.Print(scope.Name)
			}
			fmt.Println()
		}

		// Print imports
		if len(chunk.Context.Imports) > 0 {
			fmt.Print("Imports: ")
			for i, imp := range chunk.Context.Imports {
				if i > 0 {
					fmt.Print(", ")
				}
				fmt.Print(imp.Name)
			}
			fmt.Println()
		}

		fmt.Println()
		fmt.Println("--- Raw Text ---")
		fmt.Println(chunk.Text)
		fmt.Println()

		fmt.Println("--- Contextualized Text ---")
		fmt.Println(chunk.ContextualizedText)
		fmt.Println()
	}
}
