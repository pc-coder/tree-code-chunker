// Package codechunk provides AST-aware code chunking for semantic search and RAG pipelines.
//
// It uses tree-sitter to split source code at semantic boundaries (functions, classes, methods)
// rather than arbitrary character limits. Each chunk includes rich context: scope chain,
// imports, siblings, and entity signatures.
//
// # Features
//
//   - AST-aware: Splits at semantic boundaries, never mid-function
//   - Rich context: Scope chain, imports, siblings, entity signatures
//   - Contextualized text: Pre-formatted for embedding models
//   - Multi-language: TypeScript, JavaScript, Python, Rust, Go, Java
//   - Batch processing: Process entire codebases with controlled concurrency
//   - Streaming: Process large files incrementally
//
// # Basic Usage
//
//	chunks, err := codechunk.Chunk("src/user.go", sourceCode, nil)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, c := range chunks {
//	    fmt.Println(c.Text)
//	    fmt.Println(c.Context.Scope)
//	    fmt.Println(c.Context.Entities)
//	}
package codechunk

import (
	"context"
	"strings"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

// Chunk chunks source code into pieces with semantic context.
//
// This is the main entry point for the code-chunk library. It takes source code
// and returns an array of chunks, each with contextual information about the
// code's structure.
func Chunk(filepath string, code string, opts *ChunkOptions) ([]CodeChunk, error) {
	options := ChunkOptions{}
	if opts != nil {
		options = *opts
	}
	return chunkFile(filepath, []byte(code), options)
}

// ChunkBytes is like Chunk but accepts []byte instead of string.
func ChunkBytes(filepath string, code []byte, opts *ChunkOptions) ([]CodeChunk, error) {
	options := ChunkOptions{}
	if opts != nil {
		options = *opts
	}
	return chunkFile(filepath, code, options)
}

// chunkFile is the internal implementation
func chunkFile(filepath string, code []byte, opts ChunkOptions) ([]CodeChunk, error) {
	// Detect language
	lang := opts.Language
	if lang == "" {
		lang = DetectLanguage(filepath)
	}
	if lang == "" {
		return nil, ErrUnsupportedLanguage
	}

	// Parse the code
	parseResult, err := parse(code, lang)
	if err != nil {
		return nil, err
	}

	// Extract entities
	entities := extractEntities(parseResult.Tree.RootNode(), lang, code)

	// Build scope tree
	scopeTree := buildScopeTree(entities)

	// Chunk the code
	chunks, err := chunkCode(
		parseResult.Tree.RootNode(),
		code,
		scopeTree,
		lang,
		opts,
		filepath,
	)
	if err != nil {
		return nil, err
	}

	// Attach parse error to chunks if present
	if parseResult.Error != nil {
		for i := range chunks {
			chunks[i].Context.ParseError = parseResult.Error
		}
	}

	return chunks, nil
}

// chunkCode chunks source code into pieces with context
func chunkCode(
	rootNode interface{},
	code []byte,
	scopeTree *ScopeTree,
	lang Language,
	opts ChunkOptions,
	filepath string,
) ([]CodeChunk, error) {
	// Verify rootNode is a valid tree-sitter node
	_, ok := rootNode.(*sitter.Node)
	if !ok {
		return nil, ErrParseFailed
	}

	// Apply defaults
	if opts.MaxChunkSize == 0 {
		opts.MaxChunkSize = 1500
	}
	if opts.ContextMode == "" {
		opts.ContextMode = ContextModeFull
	}
	if opts.SiblingDetail == "" {
		opts.SiblingDetail = SiblingDetailSignatures
	}
	if opts.OverlapLines == 0 {
		opts.OverlapLines = 10
	}

	maxSize := opts.MaxChunkSize

	// Preprocess NWS cumulative sum
	cumsum := preprocessNwsCumsum(code)

	// Get root's children
	children := getNodeChildren(rootNode)

	// Assign nodes to windows
	rawWindows := greedyAssignWindows(children, code, cumsum, maxSize)

	// Merge adjacent windows
	mergedWindows := mergeAdjacentWindows(rawWindows, maxSize)

	totalChunks := len(mergedWindows)

	// Rebuild text for all windows
	rebuiltTexts := make([]*rebuiltText, len(mergedWindows))
	for i, window := range mergedWindows {
		rebuiltTexts[i] = rebuildText(window, code)
	}

	// Build chunks
	chunks := make([]CodeChunk, len(mergedWindows))
	for i, text := range rebuiltTexts {
		var ctx ChunkContext
		if opts.ContextMode == ContextModeNone {
			ctx = ChunkContext{
				Scope:    []EntityInfo{},
				Entities: []ChunkEntityInfo{},
				Siblings: []SiblingInfo{},
				Imports:  []ImportInfo{},
			}
		} else {
			ctx = buildChunkContext(text, scopeTree, opts, filepath, lang)
		}

		var overlapText string
		if opts.OverlapLines > 0 && i > 0 {
			prevText := rebuiltTexts[i-1]
			if prevText != nil && prevText.text != "" {
				prevLines := strings.Split(prevText.text, "\n")
				overlapLineCount := opts.OverlapLines
				if overlapLineCount > len(prevLines) {
					overlapLineCount = len(prevLines)
				}
				overlapText = strings.Join(prevLines[len(prevLines)-overlapLineCount:], "\n")
			}
		}

		contextualizedText := FormatChunkWithContext(text.text, ctx, overlapText)

		chunks[i] = CodeChunk{
			Text:               text.text,
			ContextualizedText: contextualizedText,
			ByteRange:          text.byteRange,
			LineRange:          text.lineRange,
			Context:            ctx,
			Index:              i,
			TotalChunks:        totalChunks,
		}
	}

	return chunks, nil
}

// ChunkStream streams chunks as they are generated.
// Useful for large files. Note: TotalChunks is -1 in streaming mode.
func ChunkStream(filepath string, code string, opts *ChunkOptions) (<-chan CodeChunk, error) {
	options := ChunkOptions{}
	if opts != nil {
		options = *opts
	}

	lang := options.Language
	if lang == "" {
		lang = DetectLanguage(filepath)
	}
	if lang == "" {
		return nil, ErrUnsupportedLanguage
	}

	parseResult, err := parseString(code, lang)
	if err != nil {
		return nil, err
	}

	entities := extractEntities(parseResult.Tree.RootNode(), lang, []byte(code))
	scopeTree := buildScopeTree(entities)

	ch := make(chan CodeChunk)

	go func() {
		defer close(ch)

		if options.MaxChunkSize == 0 {
			options.MaxChunkSize = 1500
		}
		if options.ContextMode == "" {
			options.ContextMode = ContextModeFull
		}
		if options.SiblingDetail == "" {
			options.SiblingDetail = SiblingDetailSignatures
		}
		if options.OverlapLines == 0 {
			options.OverlapLines = 10
		}

		maxSize := options.MaxChunkSize
		cumsum := preprocessNwsCumsum([]byte(code))
		children := getNodeChildren(parseResult.Tree.RootNode())
		rawWindows := greedyAssignWindows(children, []byte(code), cumsum, maxSize)
		mergedWindows := mergeAdjacentWindows(rawWindows, maxSize)

		var prevText string
		for i, window := range mergedWindows {
			text := rebuildText(window, []byte(code))

			var ctx ChunkContext
			if options.ContextMode == ContextModeNone {
				ctx = ChunkContext{
					Scope:    []EntityInfo{},
					Entities: []ChunkEntityInfo{},
					Siblings: []SiblingInfo{},
					Imports:  []ImportInfo{},
				}
			} else {
				ctx = buildChunkContext(text, scopeTree, options, filepath, lang)
			}

			var overlapText string
			if options.OverlapLines > 0 && prevText != "" {
				prevLines := strings.Split(prevText, "\n")
				overlapLineCount := options.OverlapLines
				if overlapLineCount > len(prevLines) {
					overlapLineCount = len(prevLines)
				}
				overlapText = strings.Join(prevLines[len(prevLines)-overlapLineCount:], "\n")
			}

			contextualizedText := FormatChunkWithContext(text.text, ctx, overlapText)

			ch <- CodeChunk{
				Text:               text.text,
				ContextualizedText: contextualizedText,
				ByteRange:          text.byteRange,
				LineRange:          text.lineRange,
				Context:            ctx,
				Index:              i,
				TotalChunks:        -1,
			}

			prevText = text.text
		}
	}()

	return ch, nil
}

// ChunkBatch processes multiple files concurrently with error handling per file.
func ChunkBatch(files []FileInput, opts *BatchOptions) []BatchResult {
	return ChunkBatchWithContext(context.Background(), files, opts)
}

// ChunkBatchWithContext processes multiple files with context for cancellation.
func ChunkBatchWithContext(ctx context.Context, files []FileInput, opts *BatchOptions) []BatchResult {
	if len(files) == 0 {
		return []BatchResult{}
	}

	options := BatchOptions{}
	if opts != nil {
		options = *opts
	}

	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	results := make([]BatchResult, len(files))
	work := make(chan int, len(files))
	for i := range files {
		work <- i
	}
	close(work)

	var completed int
	var mu sync.Mutex

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case idx, ok := <-work:
					if !ok {
						return
					}

					file := files[idx]
					fileOpts := options.ChunkOptions
					if file.Options != nil {
						if file.Options.MaxChunkSize > 0 {
							fileOpts.MaxChunkSize = file.Options.MaxChunkSize
						}
						if file.Options.ContextMode != "" {
							fileOpts.ContextMode = file.Options.ContextMode
						}
						if file.Options.SiblingDetail != "" {
							fileOpts.SiblingDetail = file.Options.SiblingDetail
						}
						if file.Options.Language != "" {
							fileOpts.Language = file.Options.Language
						}
						if file.Options.OverlapLines > 0 {
							fileOpts.OverlapLines = file.Options.OverlapLines
						}
						fileOpts.FilterImports = file.Options.FilterImports
					}

					chunks, err := chunkFile(file.Filepath, []byte(file.Code), fileOpts)

					if err != nil {
						results[idx] = BatchResult{
							Filepath: file.Filepath,
							Chunks:   nil,
							Error:    err,
						}
					} else {
						results[idx] = BatchResult{
							Filepath: file.Filepath,
							Chunks:   chunks,
							Error:    nil,
						}
					}

					mu.Lock()
					completed++
					if options.OnProgress != nil {
						options.OnProgress(completed, len(files), file.Filepath, err == nil)
					}
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait()

	return results
}

// ChunkBatchStream streams batch results as files complete processing.
func ChunkBatchStream(files []FileInput, opts *BatchOptions) <-chan BatchResult {
	return ChunkBatchStreamWithContext(context.Background(), files, opts)
}

// ChunkBatchStreamWithContext streams batch results with context for cancellation.
func ChunkBatchStreamWithContext(ctx context.Context, files []FileInput, opts *BatchOptions) <-chan BatchResult {
	ch := make(chan BatchResult)

	if len(files) == 0 {
		close(ch)
		return ch
	}

	options := BatchOptions{}
	if opts != nil {
		options = *opts
	}

	concurrency := options.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}

	go func() {
		defer close(ch)

		work := make(chan FileInput, len(files))
		for _, file := range files {
			work <- file
		}
		close(work)

		var completed int
		var mu sync.Mutex
		total := len(files)

		var wg sync.WaitGroup
		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for {
					select {
					case <-ctx.Done():
						return
					case file, ok := <-work:
						if !ok {
							return
						}

						fileOpts := options.ChunkOptions
						if file.Options != nil {
							if file.Options.MaxChunkSize > 0 {
								fileOpts.MaxChunkSize = file.Options.MaxChunkSize
							}
							if file.Options.ContextMode != "" {
								fileOpts.ContextMode = file.Options.ContextMode
							}
							if file.Options.SiblingDetail != "" {
								fileOpts.SiblingDetail = file.Options.SiblingDetail
							}
							if file.Options.Language != "" {
								fileOpts.Language = file.Options.Language
							}
							if file.Options.OverlapLines > 0 {
								fileOpts.OverlapLines = file.Options.OverlapLines
							}
							fileOpts.FilterImports = file.Options.FilterImports
						}

						chunks, err := chunkFile(file.Filepath, []byte(file.Code), fileOpts)

						var result BatchResult
						if err != nil {
							result = BatchResult{
								Filepath: file.Filepath,
								Chunks:   nil,
								Error:    err,
							}
						} else {
							result = BatchResult{
								Filepath: file.Filepath,
								Chunks:   chunks,
								Error:    nil,
							}
						}

						mu.Lock()
						completed++
						if options.OnProgress != nil {
							options.OnProgress(completed, total, file.Filepath, err == nil)
						}
						mu.Unlock()

						select {
						case <-ctx.Done():
							return
						case ch <- result:
						}
					}
				}
			}()
		}

		wg.Wait()
	}()

	return ch
}

// FormatChunkWithContext formats chunk text with semantic context prepended.
func FormatChunkWithContext(text string, ctx ChunkContext, overlapText string) string {
	parts := make([]string, 0)

	if ctx.Filepath != "" {
		relPath := getLastPathSegments(ctx.Filepath, 3)
		parts = append(parts, "# "+relPath)
	}

	if len(ctx.Scope) > 0 {
		names := make([]string, len(ctx.Scope))
		for i, s := range ctx.Scope {
			names[i] = s.Name
		}
		for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
			names[i], names[j] = names[j], names[i]
		}
		scopePath := strings.Join(names, " > ")
		parts = append(parts, "# Scope: "+scopePath)
	}

	signatures := make([]string, 0)
	for _, e := range ctx.Entities {
		if e.Signature != "" && e.Type != EntityTypeImport {
			signatures = append(signatures, e.Signature)
		}
	}
	if len(signatures) > 0 {
		parts = append(parts, "# Defines: "+strings.Join(signatures, ", "))
	}

	if len(ctx.Imports) > 0 {
		importNames := make([]string, 0)
		for i, imp := range ctx.Imports {
			if i >= 10 {
				break
			}
			importNames = append(importNames, imp.Name)
		}
		parts = append(parts, "# Uses: "+strings.Join(importNames, ", "))
	}

	beforeSiblings := make([]string, 0)
	afterSiblings := make([]string, 0)
	for _, s := range ctx.Siblings {
		if s.Position == "before" {
			beforeSiblings = append(beforeSiblings, s.Name)
		} else if s.Position == "after" {
			afterSiblings = append(afterSiblings, s.Name)
		}
	}

	if len(beforeSiblings) > 0 {
		parts = append(parts, "# After: "+strings.Join(beforeSiblings, ", "))
	}
	if len(afterSiblings) > 0 {
		parts = append(parts, "# Before: "+strings.Join(afterSiblings, ", "))
	}

	if len(parts) > 0 {
		parts = append(parts, "")
	}

	if overlapText != "" {
		parts = append(parts, "# ...")
		parts = append(parts, overlapText)
		parts = append(parts, "# ---")
	}

	parts = append(parts, text)

	return strings.Join(parts, "\n")
}

func getLastPathSegments(path string, n int) string {
	parts := strings.Split(path, "/")
	if len(parts) <= n {
		return path
	}
	return strings.Join(parts[len(parts)-n:], "/")
}

// buildChunkContext builds chunk context from scope tree
func buildChunkContext(text *rebuiltText, scopeTree *ScopeTree, opts ChunkOptions, filepath string, lang Language) ChunkContext {
	byteRange := text.byteRange

	entities := getEntitiesInRange(byteRange, scopeTree)
	scopeChain := getScopeForRange(byteRange, scopeTree)
	siblings := getSiblings(byteRange, scopeTree, opts.SiblingDetail, 3)
	imports := getRelevantImports(entities, scopeTree, opts.FilterImports)

	return ChunkContext{
		Filepath: filepath,
		Language: lang,
		Scope:    scopeChain,
		Entities: entities,
		Siblings: siblings,
		Imports:  imports,
	}
}

func getScopeForRange(byteRange ByteRange, scopeTree *ScopeTree) []EntityInfo {
	scopeNode := findScopeAtOffset(scopeTree, byteRange.Start)
	if scopeNode == nil {
		return []EntityInfo{}
	}

	scopeChain := make([]EntityInfo, 0)
	scopeChain = append(scopeChain, EntityInfo{
		Name:      scopeNode.Entity.Name,
		Type:      scopeNode.Entity.Type,
		Signature: scopeNode.Entity.Signature,
	})

	ancestors := getAncestorChain(scopeNode)
	for _, ancestor := range ancestors {
		scopeChain = append(scopeChain, EntityInfo{
			Name:      ancestor.Entity.Name,
			Type:      ancestor.Entity.Type,
			Signature: ancestor.Entity.Signature,
		})
	}

	return scopeChain
}

func getEntitiesInRange(byteRange ByteRange, scopeTree *ScopeTree) []ChunkEntityInfo {
	entities := make([]ChunkEntityInfo, 0)

	for _, entity := range scopeTree.AllEntities {
		if entity.ByteRange.Start < byteRange.End && entity.ByteRange.End > byteRange.Start {
			isPartial := entity.ByteRange.Start < byteRange.Start || entity.ByteRange.End > byteRange.End

			entityInfo := ChunkEntityInfo{
				Name:      entity.Name,
				Type:      entity.Type,
				Signature: entity.Signature,
				Docstring: entity.Docstring,
				LineRange: &entity.LineRange,
				IsPartial: isPartial,
			}
			entities = append(entities, entityInfo)
		}
	}

	return entities
}

func getSiblings(byteRange ByteRange, scopeTree *ScopeTree, detail SiblingDetail, maxSiblings int) []SiblingInfo {
	if detail == SiblingDetailNone {
		return []SiblingInfo{}
	}

	siblings := make([]SiblingInfo, 0)
	beforeCount := 0
	afterCount := 0

	for _, entity := range scopeTree.AllEntities {
		if entity.Type == EntityTypeImport || entity.Type == EntityTypeExport {
			continue
		}

		if entity.ByteRange.End <= byteRange.Start && beforeCount < maxSiblings {
			siblings = append(siblings, SiblingInfo{
				Name:     entity.Name,
				Type:     entity.Type,
				Position: "before",
				Distance: beforeCount + 1,
			})
			beforeCount++
		}

		if entity.ByteRange.Start >= byteRange.End && afterCount < maxSiblings {
			siblings = append(siblings, SiblingInfo{
				Name:     entity.Name,
				Type:     entity.Type,
				Position: "after",
				Distance: afterCount + 1,
			})
			afterCount++
		}
	}

	return siblings
}

func getRelevantImports(entities []ChunkEntityInfo, scopeTree *ScopeTree, filterImports bool) []ImportInfo {
	imports := make([]ImportInfo, 0)

	for _, imp := range scopeTree.Imports {
		source := ""
		if imp.Source != nil {
			source = *imp.Source
		}

		if !filterImports {
			imports = append(imports, ImportInfo{
				Name:   imp.Name,
				Source: source,
			})
			continue
		}

		for _, entity := range entities {
			if entity.Name == imp.Name || strings.Contains(entity.Signature, imp.Name) {
				imports = append(imports, ImportInfo{
					Name:   imp.Name,
					Source: source,
				})
				break
			}
		}
	}

	return imports
}

// Chunker is a reusable chunker instance with default options.
type Chunker struct {
	options ChunkOptions
}

// NewChunker creates a new Chunker with the given default options.
func NewChunker(opts *ChunkOptions) *Chunker {
	options := ChunkOptions{}
	if opts != nil {
		options = *opts
	}
	return &Chunker{options: options}
}

// Chunk chunks source code using this chunker's default options.
func (c *Chunker) Chunk(filepath string, code string, opts *ChunkOptions) ([]CodeChunk, error) {
	options := c.options
	if opts != nil {
		if opts.MaxChunkSize > 0 {
			options.MaxChunkSize = opts.MaxChunkSize
		}
		if opts.ContextMode != "" {
			options.ContextMode = opts.ContextMode
		}
		if opts.SiblingDetail != "" {
			options.SiblingDetail = opts.SiblingDetail
		}
		if opts.Language != "" {
			options.Language = opts.Language
		}
		if opts.OverlapLines > 0 {
			options.OverlapLines = opts.OverlapLines
		}
		if opts.FilterImports {
			options.FilterImports = opts.FilterImports
		}
	}
	return Chunk(filepath, code, &options)
}
