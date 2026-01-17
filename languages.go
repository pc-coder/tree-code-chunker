package codechunk

import (
	"path/filepath"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/tsx"
)

// LanguageExtensions maps file extensions to supported languages
var LanguageExtensions = map[string]Language{
	".ts":   LanguageTypeScript,
	".tsx":  LanguageTypeScript,
	".mts":  LanguageTypeScript,
	".cts":  LanguageTypeScript,
	".js":   LanguageJavaScript,
	".jsx":  LanguageJavaScript,
	".mjs":  LanguageJavaScript,
	".cjs":  LanguageJavaScript,
	".py":   LanguagePython,
	".pyi":  LanguagePython,
	".rs":   LanguageRust,
	".go":   LanguageGo,
	".java": LanguageJava,
}

// DetectLanguage detects the programming language from a file path based on its extension.
// Returns empty string if the language is not supported.
func DetectLanguage(path string) Language {
	ext := strings.ToLower(filepath.Ext(path))
	if lang, ok := LanguageExtensions[ext]; ok {
		return lang
	}
	return ""
}

// IsLanguageSupported returns true if the language is supported.
func IsLanguageSupported(lang Language) bool {
	switch lang {
	case LanguageTypeScript, LanguageJavaScript,
		LanguagePython, LanguageRust,
		LanguageGo, LanguageJava:
		return true
	default:
		return false
	}
}

// grammarCache caches loaded tree-sitter languages
var (
	grammarCache = make(map[Language]*sitter.Language)
	grammarMutex sync.RWMutex
)

// getLanguageGrammar returns the tree-sitter language grammar for the given language
func getLanguageGrammar(lang Language) *sitter.Language {
	grammarMutex.RLock()
	if grammar, ok := grammarCache[lang]; ok {
		grammarMutex.RUnlock()
		return grammar
	}
	grammarMutex.RUnlock()

	grammarMutex.Lock()
	defer grammarMutex.Unlock()

	// Double-check after acquiring write lock
	if grammar, ok := grammarCache[lang]; ok {
		return grammar
	}

	var grammar *sitter.Language
	switch lang {
	case LanguageTypeScript:
		grammar = tsx.GetLanguage()
	case LanguageJavaScript:
		grammar = javascript.GetLanguage()
	case LanguagePython:
		grammar = python.GetLanguage()
	case LanguageRust:
		grammar = rust.GetLanguage()
	case LanguageGo:
		grammar = golang.GetLanguage()
	case LanguageJava:
		grammar = java.GetLanguage()
	default:
		return nil
	}

	grammarCache[lang] = grammar
	return grammar
}

// ClearGrammarCache clears the grammar cache (useful for testing)
func ClearGrammarCache() {
	grammarMutex.Lock()
	defer grammarMutex.Unlock()
	grammarCache = make(map[Language]*sitter.Language)
}
