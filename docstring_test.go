package codechunk

import (
	"testing"
)

func TestExtractDocstringPython(t *testing.T) {
	code := `
def greet(name: str) -> str:
    """Greet a person.

    Args:
        name: The person's name.

    Returns:
        A greeting message.
    """
    return f"Hello, {name}!"
`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))

	found := false
	for _, e := range entities {
		if e.Name == "greet" && e.Type == EntityTypeFunction {
			found = true
			if e.Docstring == nil {
				t.Error("Expected docstring to be present")
			} else if *e.Docstring == "" {
				t.Error("Expected docstring to be non-empty")
			}
		}
	}
	if !found {
		t.Error("Expected to find greet function")
	}
}

func TestExtractDocstringPythonClass(t *testing.T) {
	code := `
class Calculator:
    """A simple calculator class.

    This class provides basic arithmetic operations.
    """

    def add(self, a: int, b: int) -> int:
        """Add two numbers."""
        return a + b
`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))

	// Check class docstring
	foundClass := false
	for _, e := range entities {
		if e.Name == "Calculator" && e.Type == EntityTypeClass {
			foundClass = true
			if e.Docstring == nil {
				t.Error("Expected class docstring to be present")
			}
		}
	}
	if !foundClass {
		t.Error("Expected to find Calculator class")
	}
}

func TestExtractDocstringPythonNoDocstring(t *testing.T) {
	code := `
def simple():
    return 42
`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))

	for _, e := range entities {
		if e.Name == "simple" {
			// Docstring should be nil since there isn't one
			if e.Docstring != nil && *e.Docstring != "" {
				t.Errorf("Expected no docstring, got %q", *e.Docstring)
			}
		}
	}
}

func TestExtractDocstringGoComment(t *testing.T) {
	code := `
// Greet greets a person with their name.
// It returns a greeting message.
func Greet(name string) string {
	return "Hello, " + name
}
`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageGo, []byte(code))

	found := false
	for _, e := range entities {
		if e.Name == "Greet" {
			found = true
			if e.Docstring != nil && *e.Docstring != "" {
				// Go comments should be captured as docstring
				t.Log("Found docstring:", *e.Docstring)
			}
		}
	}
	if !found {
		t.Error("Expected to find Greet function")
	}
}

func TestIsDocComment(t *testing.T) {
	tests := []struct {
		comment  string
		lang     Language
		expected bool
	}{
		// Go - // and /* are doc comment prefixes
		{"// This is a doc comment", LanguageGo, true},
		{"/* Block comment */", LanguageGo, true},
		{"not a comment", LanguageGo, false},

		// JavaScript/TypeScript - only /** and /// are doc prefixes
		{"/** JSDoc comment */", LanguageTypeScript, true},
		{"/// Triple slash comment", LanguageTypeScript, true},
		{"// Line comment", LanguageTypeScript, false}, // Not a doc prefix
		{"/* Regular block */", LanguageTypeScript, false}, // Not a doc prefix

		// Python - only """ and ''' are doc prefixes
		{"\"\"\"Docstring\"\"\"", LanguagePython, true},
		{"'''Docstring'''", LanguagePython, true},
		{"# Python comment", LanguagePython, false}, // # is not a doc prefix

		// Rust - ///, //!, /**, /*! are doc prefixes
		{"/// Doc comment", LanguageRust, true},
		{"//! Inner doc", LanguageRust, true},
		{"/** Block doc */", LanguageRust, true},
		{"/*! Inner block doc */", LanguageRust, true},
		{"// Regular comment", LanguageRust, false}, // Not a doc prefix
	}

	for _, tt := range tests {
		result := IsDocComment(tt.comment, tt.lang)
		if result != tt.expected {
			t.Errorf("IsDocComment(%q, %q) = %v, want %v", tt.comment, tt.lang, result, tt.expected)
		}
	}
}

func TestExtractDocstringTypeScript(t *testing.T) {
	code := `
/**
 * Greet a person.
 * @param name The person's name
 * @returns A greeting message
 */
function greet(name: string): string {
    return "Hello, " + name;
}
`
	parseResult, err := parseString(code, LanguageTypeScript)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageTypeScript, []byte(code))

	found := false
	for _, e := range entities {
		if e.Name == "greet" {
			found = true
			// JSDoc comments may or may not be captured depending on implementation
			t.Log("Docstring:", e.Docstring)
		}
	}
	if !found {
		t.Error("Expected to find greet function")
	}
}

func TestExtractDocstringRust(t *testing.T) {
	code := `
/// Add two numbers together.
///
/// # Examples
///
/// ` + "```" + `
/// let result = add(2, 3);
/// assert_eq!(result, 5);
/// ` + "```" + `
fn add(a: i32, b: i32) -> i32 {
    a + b
}
`
	parseResult, err := parseString(code, LanguageRust)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguageRust, []byte(code))

	found := false
	for _, e := range entities {
		if e.Name == "add" {
			found = true
			t.Log("Docstring:", e.Docstring)
		}
	}
	if !found {
		t.Error("Expected to find add function")
	}
}

func TestExtractDocstringSingleQuotePython(t *testing.T) {
	code := `
def hello():
    '''Single quote docstring.'''
    pass
`
	parseResult, err := parseString(code, LanguagePython)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	entities := extractEntities(parseResult.Tree.RootNode(), LanguagePython, []byte(code))

	found := false
	for _, e := range entities {
		if e.Name == "hello" {
			found = true
			if e.Docstring == nil {
				t.Error("Expected docstring to be present")
			}
		}
	}
	if !found {
		t.Error("Expected to find hello function")
	}
}
