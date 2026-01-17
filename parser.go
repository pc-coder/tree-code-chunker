package codechunk

import (
	"context"
	"errors"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"
)

var (
	// ErrUnsupportedLanguage is returned when the language is not supported
	ErrUnsupportedLanguage = errors.New("unsupported language")
	// ErrParseFailed is returned when parsing fails
	ErrParseFailed = errors.New("parse failed")
)

// parserPool manages a pool of tree-sitter parsers
var parserPool = sync.Pool{
	New: func() interface{} {
		return sitter.NewParser()
	},
}

// getParser gets a parser from the pool
func getParser() *sitter.Parser {
	return parserPool.Get().(*sitter.Parser)
}

// putParser returns a parser to the pool
func putParser(p *sitter.Parser) {
	parserPool.Put(p)
}

// parse parses source code and returns the AST
func parse(code []byte, lang Language) (*ParseResult, error) {
	return parseWithContext(context.Background(), code, lang)
}

// parseWithContext parses source code with a context for cancellation
func parseWithContext(ctx context.Context, code []byte, lang Language) (*ParseResult, error) {
	grammar := getLanguageGrammar(lang)
	if grammar == nil {
		return nil, ErrUnsupportedLanguage
	}

	parser := getParser()
	defer putParser(parser)

	parser.SetLanguage(grammar)

	tree, err := parser.ParseCtx(ctx, nil, code)
	if err != nil {
		return nil, errors.Join(ErrParseFailed, err)
	}

	result := &ParseResult{
		Tree: tree,
	}

	// Check for parse errors
	if tree.RootNode().HasError() {
		result.Error = &ParseError{
			Message:     "parse error in source code",
			Recoverable: true, // tree-sitter recovers from errors
		}
	}

	return result, nil
}

// parseString is a convenience wrapper for parsing string code
func parseString(code string, lang Language) (*ParseResult, error) {
	return parse([]byte(code), lang)
}

// hasParseErrors checks if the tree contains any parse errors
func hasParseErrors(tree *sitter.Tree) bool {
	return tree != nil && tree.RootNode().HasError()
}
