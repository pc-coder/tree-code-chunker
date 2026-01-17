package codechunk

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// BodyDelimiters maps languages to body delimiter characters
var BodyDelimiters = map[Language]string{
	LanguageTypeScript:  "{",
	LanguageJavaScript:  "{",
	LanguagePython:      ":",
	LanguageRust:        "{",
	LanguageGo:          "{",
	LanguageJava:        "{",
}

// bodyNodeTypes are node types that represent body/block structures
var bodyNodeTypes = []string{
	"block",
	"statement_block",
	"class_body",
	"interface_body",
	"enum_body",
}

// findBodyDelimiterPos finds the position of the body delimiter in a signature
func findBodyDelimiterPos(text string, delimiter string) int {
	parenDepth := 0
	bracketDepth := 0
	angleDepth := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(text); i++ {
		char := text[i]
		prevChar := byte(0)
		if i > 0 {
			prevChar = text[i-1]
		}

		// Track string literals
		if (char == '"' || char == '\'' || char == '`') && prevChar != '\\' {
			if !inString {
				inString = true
				stringChar = char
			} else if char == stringChar {
				inString = false
				stringChar = 0
			}
			continue
		}

		if inString {
			continue
		}

		// Track nested structures
		switch char {
		case '(':
			parenDepth++
		case ')':
			parenDepth--
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
		case '<':
			if i+1 < len(text) {
				nextChar := text[i+1]
				if isIdentStart(nextChar) || nextChar == '>' || nextChar == ' ' || nextChar == '<' {
					angleDepth++
				}
			}
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
		}

		// Only match delimiter at depth 0
		if string(char) == delimiter && parenDepth == 0 && bracketDepth == 0 && angleDepth == 0 {
			return i
		}
	}

	return -1
}

func isIdentStart(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_'
}

// tryExtractSignatureFromBody extracts signature using AST body field
func tryExtractSignatureFromBody(node *sitter.Node, code []byte, lang Language) string {
	bodyNode := node.ChildByFieldName("body")
	if bodyNode == nil {
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			for _, bodyType := range bodyNodeTypes {
				if child.Type() == bodyType {
					bodyNode = child
					break
				}
			}
			if bodyNode != nil {
				break
			}
		}
	}

	if bodyNode == nil {
		return ""
	}

	signature := strings.TrimSpace(string(code[node.StartByte():bodyNode.StartByte()]))

	if lang == LanguagePython && strings.HasSuffix(signature, ":") {
		signature = signature[:len(signature)-1]
	}

	if strings.HasSuffix(signature, "=>") {
		signature = strings.TrimSpace(signature[:len(signature)-2])
	}

	return cleanSignature(signature)
}

// extractSignature extracts the signature of an entity from its AST node
func extractSignature(node *sitter.Node, entityType EntityType, lang Language, code []byte) string {
	switch entityType {
	case EntityTypeFunction, EntityTypeMethod:
		return extractFunctionSignature(node, lang, code)
	case EntityTypeClass, EntityTypeInterface:
		return extractClassSignature(node, lang, code)
	case EntityTypeType, EntityTypeEnum:
		return extractTypeSignature(node, lang, code)
	case EntityTypeImport, EntityTypeExport:
		return extractImportExportSignature(node, code)
	default:
		nodeText := string(code[node.StartByte():node.EndByte()])
		firstNewline := strings.Index(nodeText, "\n")
		if firstNewline != -1 {
			return cleanSignature(nodeText[:firstNewline])
		}
		return cleanSignature(nodeText)
	}
}

func extractFunctionSignature(node *sitter.Node, lang Language, code []byte) string {
	if sig := tryExtractSignatureFromBody(node, code, lang); sig != "" {
		return sig
	}

	nodeText := string(code[node.StartByte():node.EndByte()])
	delimiter := BodyDelimiters[lang]
	delimPos := findBodyDelimiterPos(nodeText, delimiter)

	if delimPos == -1 {
		return cleanSignature(nodeText)
	}

	return cleanSignature(strings.TrimSpace(nodeText[:delimPos]))
}

func extractClassSignature(node *sitter.Node, lang Language, code []byte) string {
	if sig := tryExtractSignatureFromBody(node, code, lang); sig != "" {
		return sig
	}

	nodeText := string(code[node.StartByte():node.EndByte()])
	delimiter := BodyDelimiters[lang]
	delimPos := findBodyDelimiterPos(nodeText, delimiter)

	if delimPos == -1 {
		firstNewline := strings.Index(nodeText, "\n")
		if firstNewline != -1 {
			return cleanSignature(nodeText[:firstNewline])
		}
		return cleanSignature(nodeText)
	}

	return cleanSignature(strings.TrimSpace(nodeText[:delimPos]))
}

func extractTypeSignature(node *sitter.Node, lang Language, code []byte) string {
	nodeText := string(code[node.StartByte():node.EndByte()])

	equalsPos := strings.Index(nodeText, "=")
	bracePos := findBodyDelimiterPos(nodeText, "{")
	colonPos := -1
	if lang == LanguagePython {
		colonPos = findBodyDelimiterPos(nodeText, ":")
	}

	delimPos := -1
	if equalsPos != -1 {
		delimPos = equalsPos
	}
	if bracePos != -1 && (delimPos == -1 || bracePos < delimPos) {
		delimPos = bracePos
	}
	if colonPos != -1 && (delimPos == -1 || colonPos < delimPos) {
		delimPos = colonPos
	}

	if delimPos == -1 {
		firstNewline := strings.Index(nodeText, "\n")
		if firstNewline != -1 {
			return cleanSignature(nodeText[:firstNewline])
		}
		return cleanSignature(nodeText)
	}

	return cleanSignature(strings.TrimSpace(nodeText[:delimPos]))
}

func extractImportExportSignature(node *sitter.Node, code []byte) string {
	nodeText := string(code[node.StartByte():node.EndByte()])
	return cleanSignature(nodeText)
}

// cleanSignature cleans up a signature string
func cleanSignature(sig string) string {
	sig = strings.ReplaceAll(sig, "\r\n", " ")
	sig = strings.ReplaceAll(sig, "\n", " ")

	result := make([]byte, 0, len(sig))
	lastWasSpace := false
	for i := 0; i < len(sig); i++ {
		c := sig[i]
		isSpace := c == ' ' || c == '\t' || c == '\r' || c == '\n'
		if isSpace {
			if !lastWasSpace {
				result = append(result, ' ')
			}
			lastWasSpace = true
		} else {
			result = append(result, c)
			lastWasSpace = false
		}
	}

	return strings.TrimSpace(string(result))
}

// extractImportSource extracts the import source path from an import AST node
func extractImportSource(node *sitter.Node, lang Language, code []byte) string {
	if sourceField := node.ChildByFieldName("source"); sourceField != nil {
		return stripQuotes(string(code[sourceField.StartByte():sourceField.EndByte()]))
	}

	switch lang {
	case LanguageTypeScript, LanguageJavaScript:
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "string" {
				return stripQuotes(string(code[child.StartByte():child.EndByte()]))
			}
		}

	case LanguagePython:
		if moduleNameField := node.ChildByFieldName("module_name"); moduleNameField != nil {
			return string(code[moduleNameField.StartByte():moduleNameField.EndByte()])
		}
		if nameField := node.ChildByFieldName("name"); nameField != nil {
			return string(code[nameField.StartByte():nameField.EndByte()])
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "dotted_name" {
				return string(code[child.StartByte():child.EndByte()])
			}
		}

	case LanguageGo:
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "import_spec" {
				if pathNode := child.ChildByFieldName("path"); pathNode != nil {
					return stripQuotes(string(code[pathNode.StartByte():pathNode.EndByte()]))
				}
				for j := 0; j < int(child.ChildCount()); j++ {
					specChild := child.Child(j)
					if specChild.Type() == "interpreted_string_literal" {
						return stripQuotes(string(code[specChild.StartByte():specChild.EndByte()]))
					}
				}
			}
			if child.Type() == "interpreted_string_literal" {
				return stripQuotes(string(code[child.StartByte():child.EndByte()]))
			}
			if child.Type() == "import_spec_list" {
				for j := 0; j < int(child.ChildCount()); j++ {
					spec := child.Child(j)
					if spec.Type() == "import_spec" {
						if pathNode := spec.ChildByFieldName("path"); pathNode != nil {
							return stripQuotes(string(code[pathNode.StartByte():pathNode.EndByte()]))
						}
					}
				}
			}
		}

	case LanguageRust:
		if argField := node.ChildByFieldName("argument"); argField != nil {
			return extractRustUsePath(argField, code)
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "scoped_identifier" || child.Type() == "identifier" || child.Type() == "use_wildcard" {
				return extractRustUsePath(child, code)
			}
		}

	case LanguageJava:
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "scoped_identifier" {
				return string(code[child.StartByte():child.EndByte()])
			}
		}
	}

	importSourceNodeTypes := []string{"string", "string_literal", "interpreted_string_literal", "source"}
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		for _, nodeType := range importSourceNodeTypes {
			if child.Type() == nodeType {
				return stripQuotes(string(code[child.StartByte():child.EndByte()]))
			}
		}
	}

	return ""
}

func extractRustUsePath(node *sitter.Node, code []byte) string {
	if node.Type() == "use_list" {
		return ""
	}

	if node.Type() == "scoped_identifier" {
		lastChild := node.Child(int(node.ChildCount()) - 1)
		if lastChild != nil && lastChild.Type() == "use_list" {
			if pathChild := node.ChildByFieldName("path"); pathChild != nil {
				return string(code[pathChild.StartByte():pathChild.EndByte()])
			}
		}
	}

	return string(code[node.StartByte():node.EndByte()])
}

func stripQuotes(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') ||
			(s[0] == '`' && s[len(s)-1] == '`') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
