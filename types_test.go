package codechunk

import (
	"testing"
)

func TestDefaultChunkOptions(t *testing.T) {
	opts := DefaultChunkOptions()

	if opts.MaxChunkSize != 1500 {
		t.Errorf("expected MaxChunkSize 1500, got %d", opts.MaxChunkSize)
	}

	if opts.ContextMode != ContextModeFull {
		t.Errorf("expected ContextMode full, got %s", opts.ContextMode)
	}

	if opts.SiblingDetail != SiblingDetailSignatures {
		t.Errorf("expected SiblingDetail signatures, got %s", opts.SiblingDetail)
	}

	if opts.FilterImports != false {
		t.Error("expected FilterImports false")
	}

	if opts.OverlapLines != 10 {
		t.Errorf("expected OverlapLines 10, got %d", opts.OverlapLines)
	}
}

func TestDefaultBatchOptions(t *testing.T) {
	opts := DefaultBatchOptions()

	if opts.Concurrency != 10 {
		t.Errorf("expected Concurrency 10, got %d", opts.Concurrency)
	}

	if opts.MaxChunkSize != 1500 {
		t.Errorf("expected MaxChunkSize 1500, got %d", opts.MaxChunkSize)
	}
}

func TestLanguageConstants(t *testing.T) {
	tests := []struct {
		lang     Language
		expected string
	}{
		{LanguageTypeScript, "typescript"},
		{LanguageJavaScript, "javascript"},
		{LanguagePython, "python"},
		{LanguageRust, "rust"},
		{LanguageGo, "go"},
		{LanguageJava, "java"},
	}

	for _, tt := range tests {
		if string(tt.lang) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.lang)
		}
	}
}

func TestEntityTypeConstants(t *testing.T) {
	tests := []struct {
		entityType EntityType
		expected   string
	}{
		{EntityTypeFunction, "function"},
		{EntityTypeMethod, "method"},
		{EntityTypeClass, "class"},
		{EntityTypeInterface, "interface"},
		{EntityTypeType, "type"},
		{EntityTypeEnum, "enum"},
		{EntityTypeImport, "import"},
		{EntityTypeExport, "export"},
	}

	for _, tt := range tests {
		if string(tt.entityType) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.entityType)
		}
	}
}

func TestContextModeConstants(t *testing.T) {
	tests := []struct {
		mode     ContextMode
		expected string
	}{
		{ContextModeNone, "none"},
		{ContextModeMinimal, "minimal"},
		{ContextModeFull, "full"},
	}

	for _, tt := range tests {
		if string(tt.mode) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.mode)
		}
	}
}

func TestSiblingDetailConstants(t *testing.T) {
	tests := []struct {
		detail   SiblingDetail
		expected string
	}{
		{SiblingDetailNone, "none"},
		{SiblingDetailNames, "names"},
		{SiblingDetailSignatures, "signatures"},
	}

	for _, tt := range tests {
		if string(tt.detail) != tt.expected {
			t.Errorf("expected %s, got %s", tt.expected, tt.detail)
		}
	}
}
