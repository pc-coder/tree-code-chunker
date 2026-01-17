package codechunk

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// commentNodeTypes are node types that represent comments
var commentNodeTypes = map[string]bool{
	"comment":               true,
	"line_comment":          true,
	"block_comment":         true,
	"documentation_comment": true,
	"string":                true,
	"string_literal":        true,
	"expression_statement":  true,
}

// docCommentPrefixes are prefixes that indicate documentation comments
var docCommentPrefixes = map[Language][]string{
	LanguageTypeScript:  {"/**", "///"},
	LanguageJavaScript:  {"/**", "///"},
	LanguagePython:      {"\"\"\"", "'''"},
	LanguageRust:        {"///", "//!", "/**", "/*!"},
	LanguageGo:          {"//", "/*"},
	LanguageJava:        {"/**", "///"},
}

// IsDocComment checks if a comment text is a documentation comment
func IsDocComment(text string, lang Language) bool {
	text = strings.TrimSpace(text)
	prefixes, ok := docCommentPrefixes[lang]
	if !ok {
		return false
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// extractDocstring extracts the documentation comment for an entity
func extractDocstring(node *sitter.Node, lang Language, code []byte) *string {
	switch lang {
	case LanguagePython:
		return extractPythonDocstring(node, code)
	default:
		return extractLeadingComment(node, lang, code)
	}
}

// extractPythonDocstring extracts Python docstrings from function/class body
func extractPythonDocstring(node *sitter.Node, code []byte) *string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "block" {
				bodyNode = child
				break
			}
		}
	}

	if bodyNode == nil {
		return nil
	}

	if bodyNode.ChildCount() == 0 {
		return nil
	}

	firstStmt := bodyNode.Child(0)
	if firstStmt == nil {
		return nil
	}

	if firstStmt.Type() == "expression_statement" && firstStmt.ChildCount() > 0 {
		strNode := firstStmt.Child(0)
		if strNode != nil && strNode.Type() == "string" {
			docstring := string(code[strNode.StartByte():strNode.EndByte()])
			docstring = strings.TrimPrefix(docstring, "\"\"\"")
			docstring = strings.TrimPrefix(docstring, "'''")
			docstring = strings.TrimSuffix(docstring, "\"\"\"")
			docstring = strings.TrimSuffix(docstring, "'''")
			docstring = strings.TrimSpace(docstring)
			if docstring != "" {
				return &docstring
			}
		}
	}

	return nil
}

// extractLeadingComment extracts leading comments before an entity
func extractLeadingComment(node *sitter.Node, lang Language, code []byte) *string {
	parent := node.Parent()
	if parent == nil {
		return nil
	}

	var nodeIndex int = -1
	for i := 0; i < int(parent.ChildCount()); i++ {
		if parent.Child(i) == node {
			nodeIndex = i
			break
		}
	}

	if nodeIndex <= 0 {
		return nil
	}

	prevSibling := parent.Child(nodeIndex - 1)
	if prevSibling == nil {
		return nil
	}

	if !commentNodeTypes[prevSibling.Type()] {
		return nil
	}

	commentText := string(code[prevSibling.StartByte():prevSibling.EndByte()])

	if !IsDocComment(commentText, lang) {
		return nil
	}

	docstring := cleanDocComment(commentText, lang)
	if docstring != "" {
		return &docstring
	}

	return nil
}

// cleanDocComment cleans up a documentation comment
func cleanDocComment(text string, lang Language) string {
	text = strings.TrimSpace(text)

	switch lang {
	case LanguageTypeScript, LanguageJavaScript, LanguageJava:
		text = strings.TrimPrefix(text, "/**")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimPrefix(text, "///")
		lines := strings.Split(text, "\n")
		cleanLines := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)
			if line != "" {
				cleanLines = append(cleanLines, line)
			}
		}
		return strings.Join(cleanLines, " ")

	case LanguageGo:
		lines := strings.Split(text, "\n")
		cleanLines := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "//")
			line = strings.TrimSpace(line)
			if line != "" {
				cleanLines = append(cleanLines, line)
			}
		}
		return strings.Join(cleanLines, " ")

	case LanguageRust:
		lines := strings.Split(text, "\n")
		cleanLines := make([]string, 0, len(lines))
		for _, line := range lines {
			line = strings.TrimSpace(line)
			line = strings.TrimPrefix(line, "///")
			line = strings.TrimPrefix(line, "//!")
			line = strings.TrimPrefix(line, "/**")
			line = strings.TrimPrefix(line, "/*!")
			line = strings.TrimSuffix(line, "*/")
			line = strings.TrimPrefix(line, "*")
			line = strings.TrimSpace(line)
			if line != "" {
				cleanLines = append(cleanLines, line)
			}
		}
		return strings.Join(cleanLines, " ")

	default:
		return text
	}
}
