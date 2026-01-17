package codechunk

import (
	"strings"
	"testing"
)

func TestExtractSignatureGo(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			`func hello() {}`,
			"func hello()",
		},
		{
			`func add(a, b int) int { return a + b }`,
			"func add(a, b int) int",
		},
		{
			`func (u *User) Greet() string { return "hi" }`,
			"func (u *User) Greet() string",
		},
		{
			`type User struct { Name string }`,
			"type User struct",
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageGo)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(tt.code))
		if len(entities) == 0 {
			t.Errorf("No entities found for %q", tt.code)
			continue
		}

		if entities[0].Signature != tt.expected {
			t.Errorf("extractSignature(%q) = %q, want %q", tt.code, entities[0].Signature, tt.expected)
		}
	}
}

func TestExtractSignatureTypeScript(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			`function hello(): void {}`,
			"function hello(): void",
		},
		{
			`function add(a: number, b: number): number { return a + b; }`,
			"function add(a: number, b: number): number",
		},
		{
			`class User { constructor() {} }`,
			"class User",
		},
		{
			`interface IUser { name: string; }`,
			"interface IUser",
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageTypeScript)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(tt.code))
		if len(entities) == 0 {
			t.Errorf("No entities found for %q", tt.code)
			continue
		}

		if entities[0].Signature != tt.expected {
			t.Errorf("extractSignature(%q) = %q, want %q", tt.code, entities[0].Signature, tt.expected)
		}
	}
}

func TestExtractSignaturePython(t *testing.T) {
	tests := []struct {
		code            string
		expectedPartial string // Signature should contain this
	}{
		{
			`def hello():
    pass`,
			"def hello()",
		},
		{
			`def add(a: int, b: int) -> int:
    return a + b`,
			"def add(a: int, b: int)",
		},
		{
			`class User:
    pass`,
			"class User",
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))
		if len(entities) == 0 {
			t.Errorf("No entities found for %q", tt.code)
			continue
		}

		if !strings.Contains(entities[0].Signature, tt.expectedPartial) {
			t.Errorf("extractSignature(%q) = %q, expected to contain %q", tt.code, entities[0].Signature, tt.expectedPartial)
		}
	}
}

func TestExtractSignatureRust(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			`fn hello() {}`,
			"fn hello()",
		},
		{
			`fn add(a: i32, b: i32) -> i32 { a + b }`,
			"fn add(a: i32, b: i32) -> i32",
		},
		{
			`struct Point { x: i32, y: i32 }`,
			"struct Point",
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageRust)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(tt.code))
		if len(entities) == 0 {
			t.Errorf("No entities found for %q", tt.code)
			continue
		}

		if entities[0].Signature != tt.expected {
			t.Errorf("extractSignature(%q) = %q, want %q", tt.code, entities[0].Signature, tt.expected)
		}
	}
}

func TestExtractSignatureJava(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{
			`class Main { void hello() {} }`,
			"class Main",
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageJava)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageJava, []byte(tt.code))
		if len(entities) == 0 {
			t.Errorf("No entities found for %q", tt.code)
			continue
		}

		if entities[0].Signature != tt.expected {
			t.Errorf("extractSignature(%q) = %q, want %q", tt.code, entities[0].Signature, tt.expected)
		}
	}
}

func TestCleanSignature(t *testing.T) {
	tests := []struct {
		input           string
		expectedPartial string // Result should contain this
	}{
		{"func hello() {\n  body\n}", "func hello()"},
		{"function test() { return 1; }", "function test()"},
		{"def hello():\n    pass", "def hello()"},
		{"fn main() { }", "fn main()"},
		{"  spaced  signature  ", "spaced signature"},
		{"multiple   spaces", "multiple spaces"},
	}

	for _, tt := range tests {
		result := cleanSignature(tt.input)
		if !strings.Contains(result, tt.expectedPartial) {
			t.Errorf("cleanSignature(%q) = %q, expected to contain %q", tt.input, result, tt.expectedPartial)
		}
	}
}

func TestStripQuotes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`"hello"`, "hello"},
		{`'hello'`, "hello"},
		{"`hello`", "hello"},
		{"hello", "hello"},
		{`""`, ""},
		{`"`, `"`},
	}

	for _, tt := range tests {
		result := stripQuotes(tt.input)
		if result != tt.expected {
			t.Errorf("stripQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractImportSource(t *testing.T) {
	// TypeScript import
	tsCode := `import { useState } from 'react';`
	parseResult, err := parseString(tsCode, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(tsCode))

	foundImport := false
	for _, e := range entities {
		if e.Type == EntityTypeImport {
			foundImport = true
			if e.Source == nil || *e.Source != "react" {
				t.Errorf("Expected import source 'react', got %v", e.Source)
			}
		}
	}
	if !foundImport {
		t.Error("Expected to find import entity")
	}
}

func TestSignatureWithComments(t *testing.T) {
	code := `// This is a comment
func hello() {
    // body
}`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(code))
	if len(entities) == 0 {
		t.Fatal("Expected at least one entity")
	}

	// Signature should not include comment
	sig := entities[0].Signature
	if sig != "func hello()" {
		t.Errorf("Expected 'func hello()', got %q", sig)
	}
}
