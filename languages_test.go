package codechunk

import (
	"testing"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		filepath string
		expected Language
	}{
		// TypeScript
		{"src/index.ts", LanguageTypeScript},
		{"src/component.tsx", LanguageTypeScript},
		{"path/to/file.ts", LanguageTypeScript},

		// JavaScript
		{"app.js", LanguageJavaScript},
		{"component.jsx", LanguageJavaScript},
		{"module.mjs", LanguageJavaScript},
		{"script.cjs", LanguageJavaScript},

		// Python
		{"main.py", LanguagePython},
		{"test_module.py", LanguagePython},

		// Rust
		{"lib.rs", LanguageRust},
		{"main.rs", LanguageRust},

		// Go
		{"main.go", LanguageGo},
		{"handler_test.go", LanguageGo},

		// Java
		{"Main.java", LanguageJava},
		{"Service.java", LanguageJava},

		// Unsupported
		{"file.txt", ""},
		{"style.css", ""},
		{"index.html", ""},
		{"config.yaml", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := DetectLanguage(tt.filepath)
		if result != tt.expected {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filepath, result, tt.expected)
		}
	}
}

func TestDetectLanguagePathVariants(t *testing.T) {
	// Test with various path formats
	tests := []struct {
		filepath string
		expected Language
	}{
		{"/absolute/path/to/file.ts", LanguageTypeScript},
		{"./relative/path/file.py", LanguagePython},
		{"file.go", LanguageGo},
		{"../parent/file.rs", LanguageRust},
		{"deeply/nested/path/to/file.java", LanguageJava},
	}

	for _, tt := range tests {
		result := DetectLanguage(tt.filepath)
		if result != tt.expected {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.filepath, result, tt.expected)
		}
	}
}

func TestIsLanguageSupported(t *testing.T) {
	tests := []struct {
		lang     Language
		expected bool
	}{
		{LanguageTypeScript, true},
		{LanguageJavaScript, true},
		{LanguagePython, true},
		{LanguageRust, true},
		{LanguageGo, true},
		{LanguageJava, true},
		{"ruby", false},
		{"cpp", false},
		{"", false},
	}

	for _, tt := range tests {
		result := IsLanguageSupported(tt.lang)
		if result != tt.expected {
			t.Errorf("IsLanguageSupported(%q) = %v, want %v", tt.lang, result, tt.expected)
		}
	}
}

func TestGetLanguageGrammar(t *testing.T) {
	// Test that we can get grammars for supported languages
	languages := []Language{
		LanguageTypeScript,
		LanguageJavaScript,
		LanguagePython,
		LanguageRust,
		LanguageGo,
		LanguageJava,
	}

	for _, lang := range languages {
		grammar := getLanguageGrammar(lang)
		if grammar == nil {
			t.Errorf("getLanguageGrammar(%q) returned nil", lang)
		}
	}

	// Test unsupported language
	grammar := getLanguageGrammar("ruby")
	if grammar != nil {
		t.Error("getLanguageGrammar(ruby) should return nil")
	}
}

func TestGetLanguageGrammarCaching(t *testing.T) {
	// Call twice to test caching
	grammar1 := getLanguageGrammar(LanguageGo)
	grammar2 := getLanguageGrammar(LanguageGo)

	// Should return the same pointer (cached)
	if grammar1 != grammar2 {
		t.Error("getLanguageGrammar should return cached grammar")
	}
}
