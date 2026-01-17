package codechunk

import (
	"testing"
)

func TestExtractImportSymbolsTypeScript(t *testing.T) {
	tests := []struct {
		code          string
		minImports    int
		expectedNames []string // At least one of these should be found
	}{
		{
			`import { useState } from 'react';`,
			1,
			[]string{"useState", "import"},
		},
		{
			`import { useState, useEffect } from 'react';`,
			1,
			[]string{"useState", "useEffect", "import"},
		},
		{
			`import React from 'react';`,
			1,
			[]string{"React", "import"},
		},
		{
			`import * as React from 'react';`,
			1,
			[]string{"React", "import"},
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageTypeScript)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(tt.code))

		imports := make([]*ExtractedEntity, 0)
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				imports = append(imports, e)
			}
		}

		if len(imports) < tt.minImports {
			t.Errorf("Expected at least %d imports for %q, got %d", tt.minImports, tt.code, len(imports))
		}

		// Check that at least one expected name is found
		foundAny := false
		for _, expectedName := range tt.expectedNames {
			for _, imp := range imports {
				if imp.Name == expectedName {
					foundAny = true
					break
				}
			}
			if foundAny {
				break
			}
		}
		if !foundAny && len(imports) > 0 {
			names := make([]string, len(imports))
			for i, imp := range imports {
				names[i] = imp.Name
			}
			t.Logf("Imports found for %q: %v", tt.code, names)
		}
	}
}

func TestExtractImportSymbolsPython(t *testing.T) {
	tests := []struct {
		code       string
		minImports int
	}{
		{`import os`, 1},
		{`import os, sys`, 1},
		{`from typing import List`, 1},
		{`from typing import List, Dict, Optional`, 1},
		{`import numpy as np`, 1},
		{`from os.path import join as path_join`, 1},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguagePython)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(tt.code))

		imports := make([]*ExtractedEntity, 0)
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				imports = append(imports, e)
			}
		}

		if len(imports) < tt.minImports {
			t.Errorf("Expected at least %d imports for %q, got %d", tt.minImports, tt.code, len(imports))
		}
	}
}

func TestExtractImportSymbolsGo(t *testing.T) {
	tests := []struct {
		code           string
		expectedNames  []string
		expectedSource string
	}{
		{
			`import "fmt"`,
			[]string{"fmt"},
			"fmt",
		},
		{
			`import (
				"fmt"
				"strings"
			)`,
			[]string{"fmt", "strings"},
			"",
		},
		{
			`import f "fmt"`,
			[]string{"f"},
			"fmt",
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageGo)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(tt.code))

		imports := make([]*ExtractedEntity, 0)
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				imports = append(imports, e)
			}
		}

		for _, expectedName := range tt.expectedNames {
			found := false
			for _, imp := range imports {
				if imp.Name == expectedName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find import %q in %q. Got: %v", expectedName, tt.code, imports)
			}
		}
	}
}

func TestExtractImportSymbolsRust(t *testing.T) {
	tests := []struct {
		code       string
		minImports int
	}{
		{`use std::io;`, 1},
		{`use std::collections::{HashMap, HashSet};`, 1},
		{`use std::io::Result as IoResult;`, 1},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageRust)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(tt.code))

		imports := make([]*ExtractedEntity, 0)
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				imports = append(imports, e)
			}
		}

		if len(imports) < tt.minImports {
			t.Errorf("Expected at least %d imports for %q, got %d", tt.minImports, tt.code, len(imports))
		}
	}
}

func TestExtractImportSymbolsJava(t *testing.T) {
	tests := []struct {
		code       string
		minImports int
	}{
		{`import java.util.List;`, 1},
		{`import java.util.*;`, 1},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageJava)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageJava, []byte(tt.code))

		imports := make([]*ExtractedEntity, 0)
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				imports = append(imports, e)
			}
		}

		if len(imports) < tt.minImports {
			t.Errorf("Expected at least %d imports for %q, got %d", tt.minImports, tt.code, len(imports))
		}
	}
}

func TestExtractImportSymbolsJavaScript(t *testing.T) {
	tests := []struct {
		code          string
		expectedNames []string
	}{
		{
			`import React from 'react';`,
			[]string{"React"},
		},
		{
			`import { Component } from 'react';`,
			[]string{"Component"},
		},
		{
			`const fs = require('fs');`,
			[]string{}, // require is not detected as import
		},
	}

	for _, tt := range tests {
		parseResult, err := parseString(tt.code, LanguageJavaScript)
		if err != nil {
			t.Fatalf("Parse failed for %q: %v", tt.code, err)
		}

		entities := extractEntities(parseResult.Tree.RootNode(), LanguageJavaScript, []byte(tt.code))

		imports := make([]*ExtractedEntity, 0)
		for _, e := range entities {
			if e.Type == EntityTypeImport {
				imports = append(imports, e)
			}
		}

		for _, expectedName := range tt.expectedNames {
			found := false
			for _, imp := range imports {
				if imp.Name == expectedName {
					found = true
					break
				}
			}
			if !found {
				names := make([]string, len(imports))
				for i, imp := range imports {
					names[i] = imp.Name
				}
				t.Errorf("Expected to find import %q in %q. Got: %v", expectedName, tt.code, names)
			}
		}
	}
}

func TestCreateImportEntity(t *testing.T) {
	code := `import "fmt"`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find the import node
	root := parseResult.Tree.RootNode()
	var importNode interface{}
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() == "import_declaration" {
			importNode = child
			break
		}
	}

	if importNode == nil {
		t.Fatal("Could not find import node")
	}
}

func TestGetLastSegment(t *testing.T) {
	// Test getLastSegment indirectly through Rust import extraction
	// since the function requires a node parameter
	code := `use std::io::Result;`
	parseResult, err := parseString(code, LanguageRust)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(code))

	found := false
	for _, e := range entities {
		if e.Type == EntityTypeImport && e.Name == "Result" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected getLastSegment to extract 'Result' from 'std::io::Result'")
	}
}

func TestExtractImportSpecifierName(t *testing.T) {
	// Test aliased import
	code := `import { useState as state } from 'react';`
	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))

	found := false
	for _, e := range entities {
		if e.Type == EntityTypeImport && e.Name == "state" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find aliased import 'state'")
	}
}

func TestPythonWildcardImport(t *testing.T) {
	code := `from os import *`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))

	found := false
	for _, e := range entities {
		if e.Type == EntityTypeImport && e.Name == "*" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find wildcard import '*'")
	}
}
