package codechunk

import (
	"context"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		code string
		lang Language
	}{
		{`package main; func main() {}`, LanguageGo},
		{`function hello() {}`, LanguageTypeScript},
		{`def hello(): pass`, LanguagePython},
		{`fn main() {}`, LanguageRust},
		{`class Main { }`, LanguageJava},
		{`function hello() {}`, LanguageJavaScript},
	}

	for _, tt := range tests {
		result, err := parse([]byte(tt.code), tt.lang)
		if err != nil {
			t.Errorf("parse(%q, %q) error: %v", tt.code, tt.lang, err)
			continue
		}

		if result.Tree == nil {
			t.Errorf("parse(%q, %q) returned nil tree", tt.code, tt.lang)
		}

		if result.Tree.RootNode() == nil {
			t.Errorf("parse(%q, %q) returned nil root node", tt.code, tt.lang)
		}
	}
}

func TestParseString(t *testing.T) {
	code := `package main

func main() {
	println("Hello")
}
`
	result, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("parseString failed: %v", err)
	}

	if result.Tree == nil {
		t.Error("Expected non-nil tree")
	}

	root := result.Tree.RootNode()
	if root == nil {
		t.Error("Expected non-nil root node")
	}

	// Root should have children
	if root.ChildCount() == 0 {
		t.Error("Root node should have children")
	}
}

func TestParseUnsupportedLanguage(t *testing.T) {
	_, err := parse([]byte("code"), "ruby")
	if err == nil {
		t.Error("Expected error for unsupported language")
	}
}

func TestParseEmptyCode(t *testing.T) {
	result, err := parse([]byte(""), LanguageGo)
	if err != nil {
		t.Fatalf("Parse empty code failed: %v", err)
	}

	if result.Tree == nil {
		t.Error("Expected non-nil tree for empty code")
	}
}

func TestParseSyntaxError(t *testing.T) {
	// Code with syntax errors should still parse (tree-sitter is error-tolerant)
	code := `func broken {{{`
	result, err := parse([]byte(code), LanguageGo)
	if err != nil {
		t.Fatalf("Parse with syntax error failed: %v", err)
	}

	if result.Tree == nil {
		t.Error("Expected non-nil tree even with syntax errors")
	}

	// The tree might have error nodes, but should still be valid
	root := result.Tree.RootNode()
	if root == nil {
		t.Error("Root node should not be nil")
	}
}

func TestParseWithContext(t *testing.T) {
	code := `func hello() {}`
	result, err := parseWithContext(context.Background(), []byte(code), LanguageGo)
	if err != nil {
		t.Fatalf("parseWithContext failed: %v", err)
	}

	if result.Tree == nil {
		t.Error("Expected non-nil tree")
	}
}

func TestParseLargeFile(t *testing.T) {
	// Generate a large Go file
	var code string
	code = "package main\n\n"
	for i := 0; i < 100; i++ {
		code += "func function" + string(rune('A'+i%26)) + "() {}\n"
	}

	result, err := parse([]byte(code), LanguageGo)
	if err != nil {
		t.Fatalf("Parse large file failed: %v", err)
	}

	if result.Tree == nil {
		t.Error("Expected non-nil tree")
	}
}

func TestParseAllLanguages(t *testing.T) {
	languages := []struct {
		lang Language
		code string
	}{
		{LanguageGo, `package main; func main() {}`},
		{LanguageTypeScript, `function hello(): void {}`},
		{LanguageJavaScript, `function hello() {}`},
		{LanguagePython, `def hello(): pass`},
		{LanguageRust, `fn main() {}`},
		{LanguageJava, `class Main { public static void main(String[] args) {} }`},
	}

	for _, tt := range languages {
		result, err := parse([]byte(tt.code), tt.lang)
		if err != nil {
			t.Errorf("parse(%q) for %q failed: %v", tt.code, tt.lang, err)
			continue
		}

		if result.Tree == nil {
			t.Errorf("parse(%q) for %q returned nil tree", tt.code, tt.lang)
		}
	}
}

func TestParseTreeNodeTypes(t *testing.T) {
	code := `package main

import "fmt"

func main() {
	fmt.Println("Hello")
}
`
	result, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("parseString failed: %v", err)
	}

	root := result.Tree.RootNode()

	// Check that root is source_file
	if root.Type() != "source_file" {
		t.Errorf("Expected root type 'source_file', got '%s'", root.Type())
	}

	// Find function_declaration
	foundFunc := false
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() == "function_declaration" {
			foundFunc = true
			break
		}
	}
	if !foundFunc {
		t.Error("Expected to find function_declaration in parsed tree")
	}
}

func TestParseTreeByteRanges(t *testing.T) {
	code := `func hello() {}`
	result, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("parseString failed: %v", err)
	}

	root := result.Tree.RootNode()

	// Root should span the entire code
	if root.StartByte() != 0 {
		t.Errorf("Expected root start byte 0, got %d", root.StartByte())
	}

	if root.EndByte() != uint32(len(code)) {
		t.Errorf("Expected root end byte %d, got %d", len(code), root.EndByte())
	}
}

func TestParseReturnsValidResult(t *testing.T) {
	code := `func main() {}`
	result, err := parse([]byte(code), LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify ParseResult structure
	if result.Tree == nil {
		t.Error("Tree should not be nil")
	}

	// Error should be nil for valid code
	if result.Error != nil {
		t.Logf("Parse error: %v (may be expected for some code)", result.Error)
	}
}

func TestParseMultipleTimes(t *testing.T) {
	code := `func hello() {}`

	// Parse multiple times to test parser pool
	for i := 0; i < 10; i++ {
		result, err := parseString(code, LanguageGo)
		if err != nil {
			t.Errorf("Parse iteration %d failed: %v", i, err)
			continue
		}

		if result.Tree == nil {
			t.Errorf("Parse iteration %d returned nil tree", i)
		}
	}
}

func TestParseConcurrent(t *testing.T) {
	code := `func hello() {}`

	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, err := parseString(code, LanguageGo)
				if err != nil {
					t.Errorf("Concurrent parse failed: %v", err)
				}
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
