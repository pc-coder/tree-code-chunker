// Package codechunk provides AST-aware code chunking for semantic search and RAG pipelines.
// It uses tree-sitter to split source code at semantic boundaries (functions, classes, methods)
// rather than arbitrary character limits.
package codechunk

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// Language represents supported programming languages for AST parsing
type Language string

const (
	LanguageTypeScript  Language = "typescript"
	LanguageJavaScript  Language = "javascript"
	LanguagePython      Language = "python"
	LanguageRust        Language = "rust"
	LanguageGo          Language = "go"
	LanguageJava        Language = "java"
)

// EntityType represents types of entities that can be extracted from source code
type EntityType string

const (
	EntityTypeFunction  EntityType = "function"
	EntityTypeMethod    EntityType = "method"
	EntityTypeClass     EntityType = "class"
	EntityTypeInterface EntityType = "interface"
	EntityTypeType      EntityType = "type"
	EntityTypeEnum      EntityType = "enum"
	EntityTypeImport    EntityType = "import"
	EntityTypeExport    EntityType = "export"
)

// LineRange represents a range of lines in the source code (0-indexed, inclusive)
type LineRange struct {
	Start int `json:"start"` // Start line (0-indexed, inclusive)
	End   int `json:"end"`   // End line (0-indexed, inclusive)
}

// ByteRange represents a range of bytes in the source code (0-indexed)
type ByteRange struct {
	Start int `json:"start"` // Start byte offset (0-indexed, inclusive)
	End   int `json:"end"`   // End byte offset (0-indexed, exclusive)
}

// ParseError represents error information from parsing
type ParseError struct {
	Message     string `json:"message"`
	Recoverable bool   `json:"recoverable"`
}

// ParseResult represents the result of parsing source code
type ParseResult struct {
	Tree  *sitter.Tree
	Error *ParseError
}

// ExtractedEntity represents an entity extracted from the AST (function, class, etc.)
type ExtractedEntity struct {
	Type      EntityType  `json:"type"`      // The type of entity
	Name      string      `json:"name"`      // Name of the entity
	Signature string      `json:"signature"` // Full signature
	Docstring *string     `json:"docstring"` // Documentation comment if present
	ByteRange ByteRange   `json:"byteRange"` // Byte range in source
	LineRange LineRange   `json:"lineRange"` // Line range in source
	Parent    *string     `json:"parent"`    // Parent entity name if nested
	Node      *sitter.Node `json:"-"`         // The underlying AST node
	Source    *string     `json:"source"`    // Import source path (only for import entities)
}

// ScopeNode represents a node in the scope tree
type ScopeNode struct {
	Entity   *ExtractedEntity `json:"entity"`   // The entity at this scope level
	Children []*ScopeNode     `json:"children"` // Child scope nodes
	Parent   *ScopeNode       `json:"-"`        // Parent scope node (excluded from JSON to avoid cycles)
}

// ScopeTree represents the tree structure of the scope hierarchy of a file
type ScopeTree struct {
	Root        []*ScopeNode       `json:"root"`        // Root scope nodes (top-level entities)
	Imports     []*ExtractedEntity `json:"imports"`     // All import entities
	Exports     []*ExtractedEntity `json:"exports"`     // All export entities
	AllEntities []*ExtractedEntity `json:"allEntities"` // Flat list of all entities
}

// ASTWindow represents a window of AST nodes for context
type ASTWindow struct {
	Nodes         []*sitter.Node // The nodes in this window
	Ancestors     []*sitter.Node // Ancestor nodes for context
	Size          int            // Size of the window in NWS characters
	IsPartialNode bool           // Whether this window contains a partial node
	LineRanges    []LineRange    // Line ranges for nodes in this window
}

// EntityInfo contains information about an entity for context
type EntityInfo struct {
	Name      string     `json:"name"`                // Name of the entity
	Type      EntityType `json:"type"`                // Type of entity
	Signature string     `json:"signature,omitempty"` // Signature if available
}

// ChunkEntityInfo contains extended entity info for entities within a chunk
type ChunkEntityInfo struct {
	Name      string     `json:"name"`                // Name of the entity
	Type      EntityType `json:"type"`                // Type of entity
	Signature string     `json:"signature,omitempty"` // Signature if available
	Docstring *string    `json:"docstring,omitempty"` // Documentation comment if present
	LineRange *LineRange `json:"lineRange,omitempty"` // Line range in source
	IsPartial bool       `json:"isPartial,omitempty"` // Whether entity spans multiple chunks
}

// SiblingInfo contains information about a sibling entity
type SiblingInfo struct {
	Name     string     `json:"name"`     // Name of the sibling
	Type     EntityType `json:"type"`     // Type of sibling
	Position string     `json:"position"` // Position relative to current chunk ("before" or "after")
	Distance int        `json:"distance"` // Distance in entities from current chunk
}

// ImportInfo contains information about an import statement
type ImportInfo struct {
	Name        string `json:"name"`                  // What is being imported
	Source      string `json:"source"`                // Source module/path
	IsDefault   bool   `json:"isDefault,omitempty"`   // Whether it's a default import
	IsNamespace bool   `json:"isNamespace,omitempty"` // Whether it's a namespace import
}

// ChunkContext contains context information for a chunk
type ChunkContext struct {
	Filepath   string            `json:"filepath,omitempty"`   // File path of the source file
	Language   Language          `json:"language,omitempty"`   // Programming language
	Scope      []EntityInfo      `json:"scope"`                // Scope chain from current to root
	Entities   []ChunkEntityInfo `json:"entities"`             // Entities within this chunk
	Siblings   []SiblingInfo     `json:"siblings"`             // Nearby sibling entities
	Imports    []ImportInfo      `json:"imports"`              // Relevant imports
	ParseError *ParseError       `json:"parseError,omitempty"` // Parse error if any
}

// CodeChunk represents a chunk of source code with context
type CodeChunk struct {
	Text              string       `json:"text"`              // The actual text content
	ContextualizedText string      `json:"contextualizedText"` // Text with semantic context prepended
	ByteRange         ByteRange    `json:"byteRange"`         // Byte range in original source
	LineRange         LineRange    `json:"lineRange"`         // Line range in original source
	Context           ChunkContext `json:"context"`           // Contextual information
	Index             int          `json:"index"`             // Index of this chunk (0-based)
	TotalChunks       int          `json:"totalChunks"`       // Total number of chunks
}

// ContextMode specifies how much context to include
type ContextMode string

const (
	ContextModeNone    ContextMode = "none"
	ContextModeMinimal ContextMode = "minimal"
	ContextModeFull    ContextMode = "full"
)

// SiblingDetail specifies level of sibling detail
type SiblingDetail string

const (
	SiblingDetailNone       SiblingDetail = "none"
	SiblingDetailNames      SiblingDetail = "names"
	SiblingDetailSignatures SiblingDetail = "signatures"
)

// ChunkOptions contains options for chunking source code
type ChunkOptions struct {
	MaxChunkSize  int           `json:"maxChunkSize,omitempty"`  // Maximum chunk size in bytes (default: 1500)
	ContextMode   ContextMode   `json:"contextMode,omitempty"`   // How much context to include (default: full)
	SiblingDetail SiblingDetail `json:"siblingDetail,omitempty"` // Level of sibling detail (default: signatures)
	FilterImports bool          `json:"filterImports,omitempty"` // Filter out import statements (default: false)
	Language      Language      `json:"language,omitempty"`      // Override language detection
	OverlapLines  int           `json:"overlapLines,omitempty"`  // Lines from previous chunk to include (default: 10)
}

// DefaultChunkOptions returns the default chunk options
func DefaultChunkOptions() ChunkOptions {
	return ChunkOptions{
		MaxChunkSize:  1500,
		ContextMode:   ContextModeFull,
		SiblingDetail: SiblingDetailSignatures,
		FilterImports: false,
		OverlapLines:  10,
	}
}

// FileInput represents input for batch processing - a single file to chunk
type FileInput struct {
	Filepath string        `json:"filepath"` // File path (used for language detection)
	Code     string        `json:"code"`     // Source code content
	Options  *ChunkOptions `json:"options"`  // Optional per-file chunking options
}

// BatchResult represents the result for a single file in batch processing
type BatchResult struct {
	Filepath string      `json:"filepath"`        // File path that was processed
	Chunks   []CodeChunk `json:"chunks"`          // Generated chunks (nil on error)
	Error    error       `json:"error,omitempty"` // The error that occurred (nil on success)
}

// BatchOptions contains options for batch processing
type BatchOptions struct {
	ChunkOptions
	Concurrency int                                            `json:"concurrency,omitempty"` // Max files to process concurrently (default: 10)
	OnProgress  func(completed, total int, filepath string, success bool) `json:"-"`       // Progress callback
}

// DefaultBatchOptions returns the default batch options
func DefaultBatchOptions() BatchOptions {
	return BatchOptions{
		ChunkOptions: DefaultChunkOptions(),
		Concurrency:  10,
	}
}
