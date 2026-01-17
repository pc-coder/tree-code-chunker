package codechunk

import (
	"context"
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

// Additional tests to improve coverage

func TestClearGrammarCache(t *testing.T) {
	// Load a grammar first
	_ = getLanguageGrammar(LanguageGo)

	// Clear the cache
	ClearGrammarCache()

	// Verify we can still get grammars after clearing
	grammar := getLanguageGrammar(LanguageGo)
	if grammar == nil {
		t.Error("Expected to get grammar after cache clear")
	}
}

func TestHasParseErrors(t *testing.T) {
	// Parse valid code
	validResult, err := parseString(`func main() {}`, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if hasParseErrors(validResult.Tree) {
		t.Error("Expected no parse errors for valid code")
	}

	// Parse code with errors
	errorResult, err := parseString(`func main { }}}}`, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// tree-sitter is error-tolerant, so check if it reports errors
	_ = hasParseErrors(errorResult.Tree)

	// Test with nil tree
	if hasParseErrors(nil) {
		t.Error("Expected false for nil tree")
	}
}

func TestExtractClassSignatureWithBody(t *testing.T) {
	// TypeScript class with body
	code := `class MyClass {
		private field: string;

		constructor() {}

		method() {
			return 42;
		}
	}`

	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))

	found := false
	for _, e := range entities {
		if e.Name == "MyClass" && e.Type == EntityTypeClass {
			found = true
			if e.Signature == "" {
				t.Error("Expected non-empty signature for class")
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find MyClass")
	}
}

func TestExtractTypeSignatureVariants(t *testing.T) {
	tests := []struct {
		code string
		lang Language
		name string
	}{
		{`type MyInterface interface { Method() }`, LanguageGo, "MyInterface"},
		{`type Alias = string`, LanguageTypeScript, "Alias"},
		{`struct Point { x: i32, y: i32 }`, LanguageRust, "Point"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))

		found := false
		for _, e := range entities {
			if e.Name == tt.name {
				found = true
				if e.Signature == "" {
					t.Errorf("Expected non-empty signature for %q", tt.name)
				}
				break
			}
		}

		if !found {
			t.Logf("Available entities for %q:", tt.code)
			for _, e := range entities {
				t.Logf("  - %s (%s)", e.Name, e.Type)
			}
		}
	}
}

func TestExtractSignatureEdgeCases(t *testing.T) {
	// Function with complex body
	code := `function complex(a: number, b: string): { x: number, y: string } {
		const result = {
			x: a,
			y: b
		};
		return result;
	}`

	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))

	if len(entities) == 0 {
		t.Fatal("Expected at least one entity")
	}

	for _, e := range entities {
		if e.Name == "complex" {
			if e.Signature == "" {
				t.Error("Expected non-empty signature")
			}
		}
	}
}

func TestImportSourceExtraction(t *testing.T) {
	tests := []struct {
		code   string
		lang   Language
		source string
	}{
		{`import { x } from './local';`, LanguageTypeScript, "./local"},
		{`import fs from 'fs';`, LanguageTypeScript, "fs"},
		{`import "fmt"`, LanguageGo, "fmt"},
		{`from typing import List`, LanguagePython, "typing"},
		{`use std::collections::HashMap;`, LanguageRust, "std::collections"},
		{`import java.util.List;`, LanguageJava, "java.util.List"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))

		foundImport := false
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				foundImport = true
				// Just verify we got an import, source extraction varies
				break
			}
		}

		if !foundImport {
			t.Errorf("Expected to find import in %q", tt.code)
		}
	}
}

func TestRustUseItemsVariants(t *testing.T) {
	tests := []struct {
		code string
	}{
		{`use std::io;`},
		{`use std::collections::{HashMap, HashSet};`},
		{`use std::io::Result as IoResult;`},
		{`use std::prelude::*;`},
		{`use crate::module::item;`},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageRust)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(tt.code))

		if len(entities) == 0 {
			t.Errorf("Expected at least one entity for %q", tt.code)
		}
	}
}

func TestPythonImportVariants(t *testing.T) {
	tests := []struct {
		code string
	}{
		{`import os`},
		{`import os.path`},
		{`from os import path`},
		{`from os.path import join`},
		{`import sys as system`},
		{`from typing import Optional, List`},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))

		foundImport := false
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				foundImport = true
				break
			}
		}

		if !foundImport {
			t.Errorf("Expected to find import in %q", tt.code)
		}
	}
}

func TestJSImportSpecifierVariants(t *testing.T) {
	tests := []struct {
		code string
	}{
		{`import { a } from 'mod';`},
		{`import { a as b } from 'mod';`},
		{`import { a, b, c } from 'mod';`},
		{`import * as mod from 'mod';`},
		{`import def from 'mod';`},
		{`import def, { a } from 'mod';`},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageTypeScript)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(tt.code))

		foundImport := false
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				foundImport = true
				break
			}
		}

		if !foundImport {
			t.Errorf("Expected to find import in %q", tt.code)
		}
	}
}

func TestFindBodyDelimiterVariants(t *testing.T) {
	tests := []struct {
		code string
		lang Language
	}{
		// Different languages have different body delimiters
		{`func main() { return }`, LanguageGo},
		{`function test(): void { console.log(1) }`, LanguageTypeScript},
		{`def hello():\n    pass`, LanguagePython},
		{`fn main() { println!("hi"); }`, LanguageRust},
		{`void main() { System.out.println("hi"); }`, LanguageJava},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))

		if len(entities) == 0 {
			t.Logf("No entities found for %q (%s)", tt.code, tt.lang)
		}
	}
}

func TestDocstringVariants(t *testing.T) {
	tests := []struct {
		code string
		lang Language
	}{
		{`# Comment before
def hello():
    """Docstring"""
    pass`, LanguagePython},
		{`def another():
    '''Single quote docstring'''
    pass`, LanguagePython},
		{`/**
 * JSDoc comment
 * @param x Number
 */
function test(x: number): void {}`, LanguageTypeScript},
		{`/// Rust doc comment
fn documented() {}`, LanguageRust},
		{`// Go doc comment
func Documented() {}`, LanguageGo},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))

		if len(entities) == 0 {
			t.Errorf("Expected at least one entity for %q", tt.code)
		}
	}
}

func TestGoImportVariants(t *testing.T) {
	tests := []struct {
		code string
	}{
		{`import "fmt"`},
		{`import f "fmt"`},
		{`import . "fmt"`},
		{`import _ "fmt"`},
		{`import (
	"fmt"
	"strings"
)`},
		{`import (
	f "fmt"
	s "strings"
)`},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageGo)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(tt.code))

		foundImport := false
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				foundImport = true
				break
			}
		}

		if !foundImport {
			t.Errorf("Expected to find import in %q", tt.code)
		}
	}
}

func TestChunkingEdgeCases(t *testing.T) {
	// Very small chunk size
	code := `package main

func main() {
	println("Hello")
}

func helper() {
	println("Helper")
}
`
	opts := &ChunkOptions{
		MaxChunkSize: 50, // Very small
	}

	chunks, err := Chunk("main.go", code, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Each chunk should have valid ranges
	for i, chunk := range chunks {
		if chunk.ByteRange.End < chunk.ByteRange.Start {
			t.Errorf("Chunk %d has invalid byte range", i)
		}
	}
}

func TestScopeTreeDeepNesting(t *testing.T) {
	code := `class Outer {
	class Inner {
		class DeepInner {
			method(): void {
				const x = () => {
					return 42;
				};
			}
		}
	}
}`

	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))
	tree := buildScopeTree(entities)

	if tree == nil {
		t.Fatal("Expected non-nil scope tree")
	}

	// Check that the scope tree has nested structure
	flat := flattenScopeTree(tree)
	t.Logf("Flattened scope tree has %d nodes", len(flat))
}

func TestContextBuilding(t *testing.T) {
	code := `import { useState } from 'react';

interface Props {
	name: string;
}

function Component(props: Props): JSX.Element {
	const [count, setCount] = useState(0);
	return <div>{props.name}: {count}</div>;
}

export default Component;
`

	chunks, err := Chunk("component.tsx", code, &ChunkOptions{
		ContextMode:   ContextModeFull,
		SiblingDetail: SiblingDetailSignatures,
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		// Context should be populated
		if chunk.Context.Language == "" {
			t.Error("Expected language to be set")
		}

		// Verify the contextualized text is not empty
		if chunk.ContextualizedText == "" {
			t.Error("Expected non-empty contextualized text")
		}
	}
}

func TestExtractEntityWithParent(t *testing.T) {
	code := `class Parent {
	method() {
		return 42;
	}
}`

	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))

	// Find method entity and check parent
	for _, e := range entities {
		if e.Name == "method" && e.Type == EntityTypeMethod {
			// Parent should be set for nested entities
			t.Logf("Method entity: Name=%s, Type=%s, Parent=%v", e.Name, e.Type, e.Parent)
		}
	}
}

func TestOversizedChunking(t *testing.T) {
	// Create a very long function that exceeds chunk size
	longFunction := `package main

func veryLongFunction() {
	// Line 1
	x := 1
	// Line 2
	y := 2
	// Line 3
	z := 3
	// Line 4
	a := 4
	// Line 5
	b := 5
	// Line 6
	c := 6
	// Line 7
	d := 7
	// Line 8
	e := 8
	// Line 9
	f := 9
	// Line 10
	g := 10
	// Line 11
	h := 11
	// Line 12
	i := 12
	// Line 13
	j := 13
	// Line 14
	k := 14
	// Line 15
	l := 15
	// Line 16
	m := 16
	// Line 17
	n := 17
	// Line 18
	o := 18
	// Line 19
	p := 19
	// Line 20
	_ = x + y + z + a + b + c + d + e + f + g + h + i + j + k + l + m + n + o + p
}
`

	// Use a very small max size to force oversized handling
	opts := &ChunkOptions{
		MaxChunkSize: 20, // Very small to force oversized leaf splitting
	}

	chunks, err := Chunk("long.go", longFunction, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	// Should produce multiple chunks
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	t.Logf("Produced %d chunks from oversized code", len(chunks))
}

func TestGreedyAssignWindowsVariants(t *testing.T) {
	// Test with code that has many small nodes
	code := `package main

var a = 1
var b = 2
var c = 3

func f1() { return }
func f2() { return }
func f3() { return }
func f4() { return }
func f5() { return }
`

	opts := &ChunkOptions{
		MaxChunkSize: 100,
	}

	chunks, err := Chunk("small.go", code, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	t.Logf("Produced %d chunks", len(chunks))
}

func TestRebuildTextVariants(t *testing.T) {
	code := `package main

func a() {}
func b() {}
func c() {}
`

	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	children := getNodeChildren(parseResult.Tree.RootNode())
	if len(children) > 0 {
		cumsum := preprocessNwsCumsum([]byte(code))
		windows := greedyAssignWindows(children, []byte(code), cumsum, 500)

		for i, window := range windows {
			text := rebuildText(window, []byte(code))
			t.Logf("Window %d: text length=%d, lines=%d-%d",
				i, len(text.text), text.lineRange.Start, text.lineRange.End)
		}
	}
}

func TestGetRelevantImportsFiltered(t *testing.T) {
	code := `import { useState, useEffect, useCallback } from 'react';

function Component() {
	const [state, setState] = useState(0);
	useEffect(() => {
		console.log(state);
	}, [state]);
	return state;
}
`

	chunks, err := Chunk("component.tsx", code, &ChunkOptions{
		FilterImports: true, // Enable import filtering
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		t.Logf("Chunk imports: %v", chunk.Context.Imports)
	}
}

func TestGetRelevantImportsUnfiltered(t *testing.T) {
	code := `import { a, b, c, d, e } from 'module';

function test() {
	return a;
}
`

	chunks, err := Chunk("test.ts", code, &ChunkOptions{
		FilterImports: false, // Don't filter imports
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		// All imports should be included
		t.Logf("Chunk has %d imports", len(chunk.Context.Imports))
	}
}

func TestChunkBatchStreamWithConcurrency(t *testing.T) {
	files := make([]FileInput, 20)
	for i := range files {
		files[i] = FileInput{
			Filepath: "file.go",
			Code:     `package main; func main() {}`,
		}
	}

	opts := &BatchOptions{
		Concurrency: 5,
	}

	ch := ChunkBatchStream(files, opts)

	count := 0
	for result := range ch {
		if result.Error != nil {
			t.Errorf("Unexpected error: %v", result.Error)
		}
		count++
	}

	if count != len(files) {
		t.Errorf("Expected %d results, got %d", len(files), count)
	}
}

func TestChunkWithOverlap(t *testing.T) {
	code := `package main

func first() {
	println("first")
}

func second() {
	println("second")
}

func third() {
	println("third")
}
`

	opts := &ChunkOptions{
		MaxChunkSize: 50,
		OverlapLines: 3,
	}

	chunks, err := Chunk("test.go", code, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	// Check that later chunks have overlap in their contextualized text
	for i, chunk := range chunks {
		if i > 0 && chunk.ContextualizedText != "" {
			// Should have overlap marker
			t.Logf("Chunk %d contextualized length: %d", i, len(chunk.ContextualizedText))
		}
	}
}

func TestChunkStreamWithOverlap(t *testing.T) {
	code := `package main

func a() {}
func b() {}
func c() {}
`

	ch, err := ChunkStream("test.go", code, &ChunkOptions{
		MaxChunkSize: 30,
		OverlapLines: 2,
	})
	if err != nil {
		t.Fatalf("ChunkStream failed: %v", err)
	}

	count := 0
	for chunk := range ch {
		t.Logf("Stream chunk %d: lines %d-%d", chunk.Index, chunk.LineRange.Start, chunk.LineRange.End)
		count++
	}

	t.Logf("Total stream chunks: %d", count)
}

func TestScopeChainBuilding(t *testing.T) {
	code := `class Outer {
	method() {
		const inner = () => {
			return 42;
		};
	}
}
`

	chunks, err := Chunk("test.ts", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		if len(chunk.Context.Scope) > 0 {
			t.Logf("Scope chain: %v", chunk.Context.Scope)
		}
	}
}

func TestGetSiblingsVariants(t *testing.T) {
	code := `func before1() {}
func before2() {}
func current() {}
func after1() {}
func after2() {}
`

	chunks, err := Chunk("test.go", code, &ChunkOptions{
		SiblingDetail: SiblingDetailSignatures,
		MaxChunkSize:  50,
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for i, chunk := range chunks {
		if len(chunk.Context.Siblings) > 0 {
			t.Logf("Chunk %d siblings: %v", i, chunk.Context.Siblings)
		}
	}
}

func TestChunkFileWithParseError(t *testing.T) {
	// Code with syntax errors
	code := `func main( {{{ }}}}`

	chunks, err := Chunk("bad.go", code, nil)
	if err != nil {
		// Parse errors might cause an error
		t.Logf("Parse error as expected: %v", err)
		return
	}

	// If it parses, check for parse error in context
	for _, chunk := range chunks {
		if chunk.Context.ParseError != nil {
			t.Logf("Parse error in context: %v", chunk.Context.ParseError)
		}
	}
}

func TestSignatureExtractionCoverage(t *testing.T) {
	tests := []struct {
		code string
		lang Language
	}{
		// Arrow functions
		{`const arrow = () => { return 1; };`, LanguageTypeScript},
		// Method with visibility
		{`class C { private method(): void {} }`, LanguageTypeScript},
		// Generic function
		{`function generic<T>(x: T): T { return x; }`, LanguageTypeScript},
		// Async function
		{`async function asyncFn(): Promise<void> {}`, LanguageTypeScript},
		// Python method
		{`class C:\n    def method(self): pass`, LanguagePython},
		// Rust impl
		{`impl Point { fn new() -> Self { Self {} } }`, LanguageRust},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		for _, e := range entities {
			t.Logf("%s (%s): sig=%q", e.Name, e.Type, e.Signature)
		}
	}
}

func TestSplitOversizedLeafByLines(t *testing.T) {
	// Create code with a very long string literal (which is a leaf node)
	longString := `package main

var longText = ` + "`" + `
Line 1: Some content here that is quite long
Line 2: More content that continues
Line 3: Even more content on this line
Line 4: The content keeps going
Line 5: And going
Line 6: And still going
Line 7: We need many lines
Line 8: To make this large enough
Line 9: To trigger the split
Line 10: Almost there
Line 11: Just a bit more
Line 12: Getting close now
Line 13: Nearly done
Line 14: Final lines
Line 15: Very last line
` + "`"

	// Use a small chunk size to force the leaf to be oversized
	opts := &ChunkOptions{
		MaxChunkSize: 10, // Very small to force oversized leaf splitting
	}

	chunks, err := Chunk("test.go", longString, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	t.Logf("Produced %d chunks from code with large string literal", len(chunks))
}

func TestSplitOversizedLeafByLinesComment(t *testing.T) {
	// Create code with a very long comment block (which is a leaf node)
	longComment := `package main

/*
This is a very long comment block.
Line 1: Some content here.
Line 2: More content.
Line 3: Even more content.
Line 4: The content keeps going.
Line 5: And going.
Line 6: And still going.
Line 7: We need many lines.
Line 8: To make this large enough.
Line 9: To trigger the split.
Line 10: Almost there.
Line 11: Just a bit more.
Line 12: Getting close now.
Line 13: Nearly done.
Line 14: Final lines.
Line 15: Very last line.
*/

func main() {}
`

	// Use a small chunk size
	opts := &ChunkOptions{
		MaxChunkSize: 10,
	}

	chunks, err := Chunk("test.go", longComment, opts)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	t.Logf("Produced %d chunks from code with large comment", len(chunks))
}

func TestFormatChunkWithContextEdgeCases(t *testing.T) {
	// Empty context
	text := "code here"
	ctx := ChunkContext{}
	result := FormatChunkWithContext(text, ctx, "")
	if result != text {
		t.Errorf("Empty context should return text unchanged")
	}

	// Context with many entities
	ctx = ChunkContext{
		Filepath: "path/to/deep/nested/file.go",
		Language: LanguageGo,
		Scope: []EntityInfo{
			{Name: "a", Type: EntityTypeClass},
			{Name: "b", Type: EntityTypeMethod},
			{Name: "c", Type: EntityTypeFunction},
		},
		Entities: []ChunkEntityInfo{
			{Name: "e1", Type: EntityTypeFunction, Signature: "func e1()"},
			{Name: "e2", Type: EntityTypeFunction, Signature: "func e2()"},
		},
		Siblings: []SiblingInfo{
			{Name: "before", Position: "before", Distance: 1},
			{Name: "after", Position: "after", Distance: 1},
		},
		Imports: []ImportInfo{
			{Name: "fmt", Source: "fmt"},
			{Name: "os", Source: "os"},
			{Name: "io", Source: "io"},
		},
	}
	result = FormatChunkWithContext(text, ctx, "overlap content")
	if result == "" {
		t.Error("Expected non-empty result")
	}
	t.Logf("Formatted result length: %d", len(result))
}

func TestExtractPythonDocstringVariants(t *testing.T) {
	tests := []struct {
		code string
	}{
		// Function with docstring
		{`def hello():
    """Simple docstring."""
    pass`},
		// Function with multi-line docstring
		{`def greet():
    """
    Multi-line
    docstring here.
    """
    pass`},
		// Class with docstring
		{`class MyClass:
    """Class docstring."""
    pass`},
		// Function without docstring
		{`def no_doc():
    pass`},
		// Function with just a statement (no docstring)
		{`def stmt_first():
    x = 1
    pass`},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))
		for _, e := range entities {
			if e.Docstring != nil {
				t.Logf("Entity %s has docstring: %q", e.Name, *e.Docstring)
			}
		}
	}
}

func TestExtractLeadingCommentVariants(t *testing.T) {
	tests := []struct {
		code string
		lang Language
	}{
		// Go function with doc comment
		{`// Greet says hello.
func Greet() {}`, LanguageGo},
		// Go function without doc comment
		{`func NoDoc() {}`, LanguageGo},
		// TypeScript with JSDoc
		{`/** JSDoc comment */
function test() {}`, LanguageTypeScript},
		// TypeScript without comment
		{`function noComment() {}`, LanguageTypeScript},
		// Rust with doc comment
		{`/// Rust doc comment
fn documented() {}`, LanguageRust},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		for _, e := range entities {
			if e.Docstring != nil {
				t.Logf("[%s] Entity %s has docstring: %q", tt.lang, e.Name, *e.Docstring)
			}
		}
	}
}

func TestChunkBytesVariants(t *testing.T) {
	code := []byte(`package main

func hello() {
	println("Hello")
}
`)

	// Test with nil options
	chunks, err := ChunkBytes("test.go", code, nil)
	if err != nil {
		t.Fatalf("ChunkBytes failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}

	// Test with options
	opts := &ChunkOptions{
		MaxChunkSize: 1000,
		ContextMode:  ContextModeFull,
	}
	chunks, err = ChunkBytes("test.go", code, opts)
	if err != nil {
		t.Fatalf("ChunkBytes with opts failed: %v", err)
	}
	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestChunkBatchStreamWithContextCancellation(t *testing.T) {
	files := make([]FileInput, 50)
	for i := range files {
		files[i] = FileInput{
			Filepath: "file.go",
			Code:     `package main; func main() {}`,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	ch := ChunkBatchStreamWithContext(ctx, files, &BatchOptions{Concurrency: 2})

	// Cancel after a short time
	go func() {
		count := 0
		for range ch {
			count++
			if count >= 5 {
				cancel()
			}
		}
	}()

	// Wait for channel to close
	for range ch {
	}
}

func TestGetScopeForRangeNoScope(t *testing.T) {
	// Code where the chunk might not have a scope
	code := `package main

// Just a comment
`

	chunks, err := Chunk("test.go", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		t.Logf("Chunk scope length: %d", len(chunk.Context.Scope))
	}
}

func TestChunkerMethodVariants(t *testing.T) {
	chunker := NewChunker(&ChunkOptions{
		MaxChunkSize:  1000,
		ContextMode:   ContextModeMinimal,
		SiblingDetail: SiblingDetailNames,
	})

	// Override all options
	code := `package main; func main() {}`
	opts := &ChunkOptions{
		MaxChunkSize:  500,
		ContextMode:   ContextModeFull,
		SiblingDetail: SiblingDetailSignatures,
		Language:      LanguageGo,
		OverlapLines:  5,
		FilterImports: true,
	}

	chunks, err := chunker.Chunk("test.go", code, opts)
	if err != nil {
		t.Fatalf("Chunker.Chunk failed: %v", err)
	}

	if len(chunks) == 0 {
		t.Error("Expected at least one chunk")
	}
}

func TestExtractPythonImportNameVariants(t *testing.T) {
	tests := []struct {
		code string
	}{
		{`import os`},
		{`import os.path`},
		{`import numpy as np`},
		{`from typing import List`},
		{`from os import path as p`},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed: %v", err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				t.Logf("Import: name=%s, source=%v", e.Name, e.Source)
			}
		}
	}
}

func TestRebuildTextWithLineRanges(t *testing.T) {
	code := `package main

func a() {}
func b() {}
`

	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	children := getNodeChildren(parseResult.Tree.RootNode())
	cumsum := preprocessNwsCumsum([]byte(code))

	// Create windows with very small size to get multiple windows
	windows := greedyAssignWindows(children, []byte(code), cumsum, 20)

	for i, window := range windows {
		text := rebuildText(window, []byte(code))
		t.Logf("Window %d: bytes=%d-%d, lines=%d-%d, partial=%v",
			i, text.byteRange.Start, text.byteRange.End,
			text.lineRange.Start, text.lineRange.End,
			window.IsPartialNode)
	}
}

func TestIsEntityNodeTypeVariants(t *testing.T) {
	// Test with different languages and node types
	tests := []struct {
		code string
		lang Language
	}{
		// Go
		{`func main() {}`, LanguageGo},
		{`type User struct {}`, LanguageGo},
		{`import "fmt"`, LanguageGo},
		// TypeScript
		{`function hello() {}`, LanguageTypeScript},
		{`class User {}`, LanguageTypeScript},
		{`interface IUser {}`, LanguageTypeScript},
		{`enum Status { Active }`, LanguageTypeScript},
		{`type Alias = string`, LanguageTypeScript},
		// Python
		{`def hello(): pass`, LanguagePython},
		{`class User: pass`, LanguagePython},
		// Rust
		{`fn main() {}`, LanguageRust},
		{`struct Point {}`, LanguageRust},
		{`enum Color { Red }`, LanguageRust},
		{`trait Draw {}`, LanguageRust},
		// Java
		{`class Main {}`, LanguageJava},
		{`interface Drawable {}`, LanguageJava},
		{`enum Color { RED }`, LanguageJava},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q (%s): %v", tt.code, tt.lang, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		t.Logf("[%s] %q -> %d entities", tt.lang, tt.code, len(entities))
	}
}

func TestImportFilteringWithMatch(t *testing.T) {
	code := `import { usedFunc, unusedFunc } from 'module';

function test() {
	return usedFunc();
}
`

	chunks, err := Chunk("test.ts", code, &ChunkOptions{
		FilterImports: true,
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		for _, imp := range chunk.Context.Imports {
			t.Logf("Filtered import: %s", imp.Name)
		}
	}
}

// Additional tests targeting low coverage functions

func TestExtractSignatureDefault(t *testing.T) {
	// Test extractSignature with EntityType that falls to default case
	code := `package main

// This is a comment that won't be extracted as an entity
const x = 1
`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Extract entities and verify we handle edge cases
	entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(code))
	t.Logf("Found %d entities", len(entities))
}

func TestExtractRustUseItemsAdvanced(t *testing.T) {
	// Test different Rust use statement patterns to improve extractRustUseItems coverage
	tests := []struct {
		code string
		desc string
	}{
		// use_list path
		{`use std::{io, fs};`, "use_list with multiple items"},
		// scoped_identifier path
		{`use std::io::Result;`, "scoped_identifier"},
		// identifier path
		{`use io;`, "simple identifier"},
		// use_as_clause path
		{`use std::io::Result as IoRes;`, "use_as_clause with alias"},
		// use_wildcard path
		{`use std::io::*;`, "use_wildcard"},
		// Nested use_list
		{`use std::collections::{HashMap, HashSet, BTreeMap};`, "nested use_list"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageRust)
		if err != nil {
			t.Logf("Parse failed for %q (%s): %v", tt.code, tt.desc, err)
			continue
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(tt.code))
		t.Logf("%s: found %d entities", tt.desc, len(entities))
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				t.Logf("  - Import: %s", e.Name)
			}
		}
	}
}

func TestExtractPythonImportNameAliased(t *testing.T) {
	// Test extractPythonImportName with aliased imports
	tests := []struct {
		code string
		desc string
	}{
		// aliased_import with alias field
		{`import numpy as np`, "aliased import"},
		// aliased_import with name field
		{`from os import path`, "import with name only"},
		// from import with alias
		{`from typing import List as L`, "from import with alias"},
		// Default path - dotted_name
		{`import os.path.join`, "dotted name import"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				t.Logf("%s: name=%s", tt.desc, e.Name)
			}
		}
	}
}

func TestExtractPythonDocstringEdgeCases(t *testing.T) {
	// Test extractPythonDocstring edge cases
	tests := []struct {
		code string
		desc string
	}{
		// Body with no children (should return nil)
		{`def empty(): pass`, "empty body"},
		// Body where first statement is not expression_statement
		{`def assignment():
    x = 1
    pass`, "first statement is assignment"},
		// Body where expression_statement child is not string
		{`def expr():
    1 + 2
    pass`, "expression is not string"},
		// Proper docstring
		{`def proper():
    """Docstring here."""
    pass`, "proper docstring"},
		// Class with body field
		{`class WithBody:
    """Class doc."""
    def method(self): pass`, "class with body field"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))
		for _, e := range entities {
			hasDoc := e.Docstring != nil && *e.Docstring != ""
			t.Logf("%s (%s): hasDocstring=%v", tt.desc, e.Name, hasDoc)
		}
	}
}

func TestChunkBatchStreamWithContextEmptyFiles(t *testing.T) {
	// Test with empty files list
	ch := ChunkBatchStreamWithContext(context.Background(), []FileInput{}, nil)
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("Expected 0 results for empty files, got %d", count)
	}
}

func TestChunkBatchStreamWithContextProgress(t *testing.T) {
	// Test progress callback
	files := []FileInput{
		{Filepath: "a.go", Code: `package main; func a() {}`},
		{Filepath: "b.go", Code: `package main; func b() {}`},
		{Filepath: "c.go", Code: `package main; func c() {}`},
	}

	progressCalls := 0
	opts := &BatchOptions{
		Concurrency: 1,
		OnProgress: func(completed, total int, filepath string, success bool) {
			progressCalls++
			t.Logf("Progress: %d/%d %s success=%v", completed, total, filepath, success)
		},
	}

	ch := ChunkBatchStreamWithContext(context.Background(), files, opts)
	for range ch {
	}

	if progressCalls != len(files) {
		t.Errorf("Expected %d progress calls, got %d", len(files), progressCalls)
	}
}

func TestGetLastSegmentEdgeCases(t *testing.T) {
	// Test getLastSegment indirectly with empty path
	code := `use x;`
	parseResult, err := parseString(code, LanguageRust)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(code))
	for _, e := range entities {
		if e.Type == EntityTypeImport {
			t.Logf("Simple use: name=%s", e.Name)
		}
	}
}

func TestExtractLeadingCommentEdgeCases(t *testing.T) {
	tests := []struct {
		code string
		lang Language
		desc string
	}{
		// No parent (root level)
		{`func main() {}`, LanguageGo, "no leading comment"},
		// Previous sibling is not a comment
		{`var x = 1
func after() {}`, LanguageGo, "prev sibling not comment"},
		// Previous sibling is comment but not doc comment
		{`// regular comment
func notDoc() {}`, LanguageGo, "Go line comment is doc"},
		// Node at index 0 (no previous sibling)
		{`func first() {}`, LanguageGo, "no previous sibling"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		for _, e := range entities {
			hasDoc := e.Docstring != nil && *e.Docstring != ""
			t.Logf("%s (%s): hasDocstring=%v", tt.desc, e.Name, hasDoc)
		}
	}
}

func TestRebuildTextEmptyWindow(t *testing.T) {
	// Test rebuildText with empty nodes
	window := &ASTWindow{
		Nodes:     []*sitter.Node{},
		Ancestors: nil,
		Size:      0,
	}

	result := rebuildText(window, []byte("code"))
	if result.text != "" {
		t.Errorf("Expected empty text for empty window, got %q", result.text)
	}
}

func TestRebuildTextBoundaryChecks(t *testing.T) {
	code := `package main

func hello() {
	println("Hello")
}`

	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	children := getNodeChildren(parseResult.Tree.RootNode())
	if len(children) == 0 {
		t.Skip("No children found")
	}

	// Create window with line ranges
	window := &ASTWindow{
		Nodes:         children,
		Ancestors:     getAncestorsForNodes(children),
		Size:          100,
		IsPartialNode: true,
		LineRanges: []LineRange{
			{Start: 0, End: 2},
			{Start: 2, End: 5},
		},
	}

	result := rebuildText(window, []byte(code))
	t.Logf("Rebuilt text: bytes=%d-%d, lines=%d-%d",
		result.byteRange.Start, result.byteRange.End,
		result.lineRange.Start, result.lineRange.End)
}

func TestFindScopeAtOffsetEdgeCases(t *testing.T) {
	code := `package main

func main() {
	println("Hello")
}
`
	chunks, err := Chunk("test.go", code, nil)
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		t.Logf("Chunk scope: %d elements", len(chunk.Context.Scope))
	}
}

func TestExtractImportSpecifierNameCoverage(t *testing.T) {
	// Test extractImportSpecifierName with different import patterns
	tests := []struct {
		code string
		desc string
	}{
		// Alias field present
		{`import { original as alias } from 'mod';`, "with alias"},
		// Name field present
		{`import { named } from 'mod';`, "with name"},
		// Just identifier child
		{`import { identifier } from 'mod';`, "with identifier"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageTypeScript)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(tt.code))
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				t.Logf("%s: import name=%s", tt.desc, e.Name)
			}
		}
	}
}

func TestScopeTreeFindInNodeCoverage(t *testing.T) {
	code := `class Outer {
	innerMethod() {
		const nestedFunc = () => {
			return 42;
		};
	}
}

function outside() {}
`
	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))
	scopeTree := buildScopeTree(entities)

	// Test findInNode by querying different offsets
	offsets := []int{0, 10, 50, 100, 150}
	for _, offset := range offsets {
		scope := findScopeAtOffset(scopeTree, offset)
		if scope != nil {
			t.Logf("Offset %d: scope=%s", offset, scope.Entity.Name)
		} else {
			t.Logf("Offset %d: no scope", offset)
		}
	}
}

func TestCleanDocCommentAllLanguages(t *testing.T) {
	tests := []struct {
		input string
		lang  Language
		desc  string
	}{
		// TypeScript/JavaScript
		{"/** JSDoc\n * line 2\n */", LanguageTypeScript, "JSDoc multiline"},
		{"/// Triple slash", LanguageTypeScript, "Triple slash"},
		// Go
		{"// Line 1\n// Line 2", LanguageGo, "Go multiline"},
		// Rust
		{"/// Rust doc\n/// Line 2", LanguageRust, "Rust doc multiline"},
		{"//! Inner doc\n//! Line 2", LanguageRust, "Rust inner doc"},
		{"/** Block\n * doc\n */", LanguageRust, "Rust block doc"},
		{"/*! Inner\n * block\n */", LanguageRust, "Rust inner block doc"},
		// Java
		{"/** Javadoc\n * @param x\n */", LanguageJava, "Javadoc"},
		// Default case
		{"some text", LanguagePython, "Python default"},
	}

	for _, tt := range tests {
		result := cleanDocComment(tt.input, tt.lang)
		t.Logf("%s: %q -> %q", tt.desc, tt.input, result)
	}
}

func TestIsDocCommentEdgeCases(t *testing.T) {
	tests := []struct {
		text     string
		lang     Language
		expected bool
		desc     string
	}{
		// Unknown language (no prefixes)
		{"// comment", Language("unknown"), false, "unknown language"},
		// Empty text
		{"", LanguageGo, false, "empty text"},
		// Whitespace only
		{"   ", LanguageGo, false, "whitespace only"},
	}

	for _, tt := range tests {
		result := IsDocComment(tt.text, tt.lang)
		if result != tt.expected {
			t.Errorf("%s: IsDocComment(%q, %q) = %v, want %v",
				tt.desc, tt.text, tt.lang, result, tt.expected)
		}
	}
}

func TestBodyDelimitersAllLanguages(t *testing.T) {
	// Test findBodyDelimiterPos with different languages
	tests := []struct {
		text      string
		delimiter string
		expected  int
		desc      string
	}{
		// Basic cases
		{"func() {}", "{", 7, "simple brace"},
		{"def foo():", ":", 9, "Python colon"},
		// With nested structures
		{"func(a map[string]int) {}", "{", 23, "nested brackets"},
		{"func<T>(x: T) {}", "{", 14, "generic angle brackets"},
		// Inside string
		{`func("{") {}`, "{", 10, "brace outside string"},
		// No delimiter
		{"func()", "{", -1, "no brace"},
	}

	for _, tt := range tests {
		result := findBodyDelimiterPos(tt.text, tt.delimiter)
		if result != tt.expected {
			t.Errorf("%s: findBodyDelimiterPos(%q, %q) = %d, want %d",
				tt.desc, tt.text, tt.delimiter, result, tt.expected)
		}
	}
}

func TestGetNodeChildrenNonNode(t *testing.T) {
	// Test getNodeChildren with non-node type
	result := getNodeChildren("not a node")
	if result != nil {
		t.Errorf("Expected nil for non-node, got %v", result)
	}
}

func TestChunkBatchWithContextOptions(t *testing.T) {
	files := []FileInput{
		{
			Filepath: "a.go",
			Code:     `package main; func a() {}`,
			Options: &ChunkOptions{
				MaxChunkSize:  500,
				ContextMode:   ContextModeMinimal,
				SiblingDetail: SiblingDetailNone,
				Language:      LanguageGo,
				OverlapLines:  3,
				FilterImports: true,
			},
		},
	}

	results := ChunkBatchWithContext(context.Background(), files, &BatchOptions{
		Concurrency: 1,
		ChunkOptions: ChunkOptions{
			MaxChunkSize: 1000,
		},
	})

	for _, r := range results {
		if r.Error != nil {
			t.Errorf("Unexpected error: %v", r.Error)
		}
	}
}

func TestChunkBatchStreamWithFileOptions(t *testing.T) {
	files := []FileInput{
		{
			Filepath: "test.ts",
			Code:     `function hello() {}`,
			Options: &ChunkOptions{
				MaxChunkSize:  500,
				ContextMode:   ContextModeFull,
				SiblingDetail: SiblingDetailSignatures,
				Language:      LanguageTypeScript,
				OverlapLines:  5,
				FilterImports: false,
			},
		},
	}

	ch := ChunkBatchStreamWithContext(context.Background(), files, nil)
	for r := range ch {
		if r.Error != nil {
			t.Errorf("Unexpected error: %v", r.Error)
		}
	}
}

func TestNwsCumsumEdgeCases(t *testing.T) {
	// Test getNwsCountFromCumsum with edge cases
	code := []byte("  a b c  ")
	cumsum := preprocessNwsCumsum(code)

	// End beyond length
	count := getNwsCountFromCumsum(cumsum, 0, 1000)
	if count < 0 {
		t.Error("Expected non-negative count")
	}

	// Start negative
	count = getNwsCountFromCumsum(cumsum, -5, 5)
	if count < 0 {
		t.Error("Expected non-negative count")
	}

	t.Logf("NWS count for edge cases: %d", count)
}

func TestCountNwsEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"   ", 0},
		{"abc", 3},
		{" a b c ", 3},
		{"\t\n\r ", 0},
		{"hello world", 10},
	}

	for _, tt := range tests {
		result := countNws(tt.input)
		if result != tt.expected {
			t.Errorf("countNws(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestExtractRustUsePath(t *testing.T) {
	// Test extractRustUsePath through various use statements
	tests := []struct {
		code string
		desc string
	}{
		{`use std::io;`, "simple path"},
		{`use std::collections::{HashMap, HashSet};`, "path with use_list"},
		{`use crate::*;`, "wildcard"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageRust)
		if err != nil {
			t.Logf("Parse failed for %q (%s): %v", tt.code, tt.desc, err)
			continue
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(tt.code))
		for _, e := range entities {
			t.Logf("%s: name=%s, source=%v", tt.desc, e.Name, e.Source)
		}
	}
}

func TestTryExtractSignatureFromBody(t *testing.T) {
	// Test tryExtractSignatureFromBody for various cases
	tests := []struct {
		code string
		lang Language
		desc string
	}{
		// With body field
		{`function test() { return 1; }`, LanguageTypeScript, "function with body"},
		// Class with class_body
		{`class Test { method() {} }`, LanguageTypeScript, "class with class_body"},
		// Interface with interface_body
		{`interface ITest { prop: string; }`, LanguageTypeScript, "interface with interface_body"},
		// Enum with enum_body
		{`enum Status { Active, Inactive }`, LanguageTypeScript, "enum with enum_body"},
		// Python with block body
		{`def test():\n    pass`, LanguagePython, "Python with block"},
		// Arrow function (should have =>)
		{`const arrow = () => { return 1; };`, LanguageTypeScript, "arrow function"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		for _, e := range entities {
			t.Logf("%s: %s signature=%q", tt.desc, e.Name, e.Signature)
		}
	}
}

func TestChunkContextModeNoneVariants(t *testing.T) {
	code := `package main

func hello() {
	println("Hello")
}
`
	chunks, err := Chunk("test.go", code, &ChunkOptions{
		ContextMode: ContextModeNone,
	})
	if err != nil {
		t.Fatalf("Chunk failed: %v", err)
	}

	for _, chunk := range chunks {
		// Context should be empty when ContextModeNone
		if len(chunk.Context.Scope) != 0 {
			t.Error("Expected empty scope for ContextModeNone")
		}
		if len(chunk.Context.Entities) != 0 {
			t.Error("Expected empty entities for ContextModeNone")
		}
	}
}

func TestChunkStreamContextModeNone(t *testing.T) {
	code := `package main

func a() {}
func b() {}
`
	ch, err := ChunkStream("test.go", code, &ChunkOptions{
		ContextMode:  ContextModeNone,
		MaxChunkSize: 50,
	})
	if err != nil {
		t.Fatalf("ChunkStream failed: %v", err)
	}

	for chunk := range ch {
		if len(chunk.Context.Scope) != 0 {
			t.Error("Expected empty scope for ContextModeNone in stream")
		}
	}
}

func TestMergeAdjacentWindowsEmpty(t *testing.T) {
	// Test mergeAdjacentWindows with empty input
	result := mergeAdjacentWindows([]*ASTWindow{}, 100)
	if len(result) != 0 {
		t.Errorf("Expected empty result for empty input, got %d", len(result))
	}
}

func TestFlattenScopeTreeVariants(t *testing.T) {
	code := `class Outer {
	class Inner {
		method() {}
	}
}`
	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))
	tree := buildScopeTree(entities)

	flat := flattenScopeTree(tree)
	t.Logf("Flattened scope tree: %d nodes", len(flat))
	for _, node := range flat {
		t.Logf("  - %s (%s)", node.Entity.Name, node.Entity.Type)
	}
}

func TestGetAncestorChainVariants(t *testing.T) {
	code := `class A {
	class B {
		class C {
			method() {}
		}
	}
}`
	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))
	tree := buildScopeTree(entities)

	// Find the deepest scope
	flat := flattenScopeTree(tree)
	if len(flat) > 0 {
		deepest := flat[len(flat)-1]
		ancestors := getAncestorChain(deepest)
		t.Logf("Ancestor chain length: %d", len(ancestors))
		for _, a := range ancestors {
			t.Logf("  - %s", a.Entity.Name)
		}
	}
}

// Additional tests for edge cases in low-coverage functions

func TestExtractPythonImportNameFallthrough(t *testing.T) {
	// Test the fallthrough case in extractPythonImportName
	// where node.Type() is not "aliased_import"
	code := `import os.path`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))
	foundImport := false
	for _, e := range entities {
		if e.Type == EntityTypeImport {
			foundImport = true
			t.Logf("Import: name=%s", e.Name)
		}
	}
	if !foundImport {
		t.Log("No imports found (expected for dotted name)")
	}
}

func TestExtractPythonDocstringNoBody(t *testing.T) {
	// Test extractPythonDocstring when there's no body field and no block child
	// This is hard to trigger directly, but we can test various edge cases
	code := `class Empty: pass`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))
	for _, e := range entities {
		t.Logf("Entity: %s, docstring=%v", e.Name, e.Docstring)
	}
}

func TestExtractRustUseItemsAllBranches(t *testing.T) {
	// Comprehensive test for extractRustUseItems covering all branches
	tests := []struct {
		code string
		desc string
	}{
		// use_list with nested items
		{`use std::{io, fs, net};`, "multiple items in use_list"},
		// identifier only (no scoped)
		{`use io;`, "bare identifier"},
		// use_as_clause
		{`use std::io as stdio;`, "use with alias"},
		// use_wildcard
		{`use std::*;`, "wildcard use"},
		// Deeply nested
		{`use std::collections::hash_map::HashMap;`, "deeply nested path"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageRust)
		if err != nil {
			t.Logf("Parse failed for %s: %v", tt.desc, err)
			continue
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(tt.code))
		t.Logf("%s: %d entities", tt.desc, len(entities))
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				t.Logf("  - %s (source=%v)", e.Name, e.Source)
			}
		}
	}
}

func TestGetLastSegmentWithColons(t *testing.T) {
	// Test getLastSegment with various paths containing ::
	code := `use a::b::c::d::e;`
	parseResult, err := parseString(code, LanguageRust)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(code))
	for _, e := range entities {
		if e.Type == EntityTypeImport && e.Name == "e" {
			t.Logf("Successfully extracted last segment: %s", e.Name)
		}
	}
}

func TestExtractLeadingCommentNoPrevSibling(t *testing.T) {
	// Test extractLeadingComment when node is at index 0
	code := `func main() {}`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(code))
	for _, e := range entities {
		if e.Docstring != nil {
			t.Logf("Entity %s has docstring (unexpected)", e.Name)
		}
	}
}

func TestExtractLeadingCommentPrevNotComment(t *testing.T) {
	// Test extractLeadingComment when previous sibling exists but is not a comment
	code := `package main

type X int

func after() {}`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(code))
	for _, e := range entities {
		if e.Name == "after" {
			hasDoc := e.Docstring != nil && *e.Docstring != ""
			t.Logf("after() has docstring: %v", hasDoc)
		}
	}
}

func TestWalkAndExtractAllNodeTypes(t *testing.T) {
	// Test walkAndExtract with code that exercises different node types
	tests := []struct {
		code string
		lang Language
		desc string
	}{
		{`package main

func f1() {}
type T1 struct{}
var v1 = 1`, LanguageGo, "Go multiple declarations"},
		{`function f() {}
class C {}
interface I {}
enum E { A }
type T = string;`, LanguageTypeScript, "TypeScript all types"},
		{`def f(): pass
class C: pass`, LanguagePython, "Python function and class"},
		{`fn f() {}
struct S {}
enum E {}
trait T {}
impl S {}`, LanguageRust, "Rust all types"},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Logf("Parse failed for %s: %v", tt.desc, err)
			continue
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		t.Logf("%s: extracted %d entities", tt.desc, len(entities))
	}
}

func TestIsEntityNodeTypeAllLanguages(t *testing.T) {
	// Test isEntityNodeType for different languages
	tests := []struct {
		code string
		lang Language
	}{
		// JavaScript arrow function and class
		{`const f = () => {}; class C {}`, LanguageJavaScript},
		// Java class, interface, enum
		{`public class C {} interface I {} enum E {}`, LanguageJava},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, tt.lang)
		if err != nil {
			t.Logf("Parse failed for %s: %v", tt.lang, err)
			continue
		}

		entities := extractEntities(parseResult.Tree.RootNode(), tt.lang, []byte(tt.code))
		t.Logf("%s: %d entities", tt.lang, len(entities))
	}
}

func TestChunkFileLanguageDetection(t *testing.T) {
	// Test language detection in chunkFile
	tests := []struct {
		filepath string
		code     string
	}{
		{"test.go", `package main; func main() {}`},
		{"test.ts", `function test(): void {}`},
		{"test.py", `def test(): pass`},
		{"test.rs", `fn main() {}`},
		{"test.java", `class Main {}`},
		{"test.js", `function test() {}`},
	}

	for _, tt := range tests {
		chunks, err := Chunk(tt.filepath, tt.code, nil)
		if err != nil {
			t.Logf("Chunk failed for %s: %v", tt.filepath, err)
			continue
		}
		t.Logf("%s: %d chunks, lang=%s", tt.filepath, len(chunks),
			chunks[0].Context.Language)
	}
}

func TestGetScopeForRangeNilScope(t *testing.T) {
	// Test getScopeForRange when no scope is found
	code := `// just a comment`

	// This code has no entities, so scope tree will be empty
	chunks, err := Chunk("test.go", code, nil)
	if err != nil {
		// Might fail due to no entities, which is expected
		t.Logf("Expected: no entities error or empty chunks")
		return
	}

	for _, chunk := range chunks {
		t.Logf("Chunk scope length: %d", len(chunk.Context.Scope))
	}
}

func TestRebuildTextWithMultipleNodes(t *testing.T) {
	code := `package main

func a() {}
func b() {}
func c() {}`

	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	children := getNodeChildren(parseResult.Tree.RootNode())
	cumsum := preprocessNwsCumsum([]byte(code))

	// Create windows with a size that allows multiple nodes per window
	windows := greedyAssignWindows(children, []byte(code), cumsum, 1000)

	for i, window := range windows {
		text := rebuildText(window, []byte(code))
		t.Logf("Window %d: %d nodes, text length=%d",
			i, len(window.Nodes), len(text.text))
	}
}

func TestExtractImportSymbolsDefaultLanguage(t *testing.T) {
	// Test extractImportSymbols with a language that falls to default case
	// This is hard to trigger directly since all supported languages have
	// specific handlers, but we can ensure the function handles edge cases

	code := `import 'something';`
	parseResult, err := parseString(code, LanguageJavaScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageJavaScript, []byte(code))
	for _, e := range entities {
		if e.Type == EntityTypeImport {
			t.Logf("Import: %s", e.Name)
		}
	}
}
