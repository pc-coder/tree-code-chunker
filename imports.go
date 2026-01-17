package codechunk

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// extractImportSymbols extracts individual import symbols from an import statement
func extractImportSymbols(node *sitter.Node, lang Language, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)
	source := extractImportSource(node, lang, code)

	switch lang {
	case LanguageTypeScript, LanguageJavaScript:
		entities = extractJSImportSymbols(node, source, code)
	case LanguagePython:
		entities = extractPythonImportSymbols(node, source, code)
	case LanguageGo:
		entities = extractGoImportSymbols(node, code)
	case LanguageRust:
		entities = extractRustImportSymbols(node, source, code)
	case LanguageJava:
		entities = extractJavaImportSymbols(node, source, code)
	default:
		entities = append(entities, createImportEntity(node, "import", source, code))
	}

	return entities
}

func extractJSImportSymbols(node *sitter.Node, source string, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)

		if child.Type() == "import_clause" {
			for j := 0; j < int(child.ChildCount()); j++ {
				clauseChild := child.Child(j)
				switch clauseChild.Type() {
				case "identifier":
					name := string(code[clauseChild.StartByte():clauseChild.EndByte()])
					entities = append(entities, createImportEntity(node, name, source, code))
				case "named_imports":
					for k := 0; k < int(clauseChild.ChildCount()); k++ {
						spec := clauseChild.Child(k)
						if spec.Type() == "import_specifier" {
							name := extractImportSpecifierName(spec, code)
							if name != "" {
								entities = append(entities, createImportEntity(node, name, source, code))
							}
						}
					}
				case "namespace_import":
					if aliasNode := clauseChild.ChildByFieldName("alias"); aliasNode != nil {
						name := string(code[aliasNode.StartByte():aliasNode.EndByte()])
						entities = append(entities, createImportEntity(node, name, source, code))
					}
				}
			}
		}
	}

	if len(entities) == 0 {
		entities = append(entities, createImportEntity(node, "import", source, code))
	}

	return entities
}

func extractImportSpecifierName(spec *sitter.Node, code []byte) string {
	if alias := spec.ChildByFieldName("alias"); alias != nil {
		return string(code[alias.StartByte():alias.EndByte()])
	}
	if name := spec.ChildByFieldName("name"); name != nil {
		return string(code[name.StartByte():name.EndByte()])
	}
	for i := 0; i < int(spec.ChildCount()); i++ {
		child := spec.Child(i)
		if child.Type() == "identifier" {
			return string(code[child.StartByte():child.EndByte()])
		}
	}
	return ""
}

func extractPythonImportSymbols(node *sitter.Node, source string, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)

	switch node.Type() {
	case "import_statement":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() == "dotted_name" || child.Type() == "aliased_import" {
				name := extractPythonImportName(child, code)
				if name != "" {
					entities = append(entities, createImportEntity(node, name, source, code))
				}
			}
		}

	case "import_from_statement":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			switch child.Type() {
			case "aliased_import":
				name := extractPythonImportName(child, code)
				if name != "" {
					entities = append(entities, createImportEntity(node, name, source, code))
				}
			case "identifier":
				name := string(code[child.StartByte():child.EndByte()])
				if name != "from" && name != "import" {
					entities = append(entities, createImportEntity(node, name, source, code))
				}
			case "wildcard_import":
				entities = append(entities, createImportEntity(node, "*", source, code))
			}
		}
	}

	if len(entities) == 0 {
		entities = append(entities, createImportEntity(node, "import", source, code))
	}

	return entities
}

func extractPythonImportName(node *sitter.Node, code []byte) string {
	if node.Type() == "aliased_import" {
		if alias := node.ChildByFieldName("alias"); alias != nil {
			return string(code[alias.StartByte():alias.EndByte()])
		}
		if name := node.ChildByFieldName("name"); name != nil {
			return string(code[name.StartByte():name.EndByte()])
		}
	}
	return string(code[node.StartByte():node.EndByte()])
}

func extractGoImportSymbols(node *sitter.Node, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)

	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		switch child.Type() {
		case "import_spec":
			name, source := extractGoImportSpec(child, code)
			entities = append(entities, createImportEntity(node, name, source, code))
		case "import_spec_list":
			for j := 0; j < int(child.ChildCount()); j++ {
				spec := child.Child(j)
				if spec.Type() == "import_spec" {
					name, source := extractGoImportSpec(spec, code)
					entities = append(entities, createImportEntity(node, name, source, code))
				}
			}
		}
	}

	if len(entities) == 0 {
		entities = append(entities, createImportEntity(node, "import", "", code))
	}

	return entities
}

func extractGoImportSpec(spec *sitter.Node, code []byte) (name string, source string) {
	if alias := spec.ChildByFieldName("name"); alias != nil {
		name = string(code[alias.StartByte():alias.EndByte()])
	}

	if path := spec.ChildByFieldName("path"); path != nil {
		source = stripQuotes(string(code[path.StartByte():path.EndByte()]))
		if name == "" {
			parts := strings.Split(source, "/")
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
	}

	if name == "" {
		name = "import"
	}

	return name, source
}

func extractRustImportSymbols(node *sitter.Node, source string, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)

	if arg := node.ChildByFieldName("argument"); arg != nil {
		extractRustUseItems(arg, source, code, &entities, node)
	}

	if len(entities) == 0 {
		entities = append(entities, createImportEntity(node, "use", source, code))
	}

	return entities
}

func extractRustUseItems(node *sitter.Node, source string, code []byte, entities *[]*ExtractedEntity, importNode *sitter.Node) {
	switch node.Type() {
	case "use_list":
		for i := 0; i < int(node.ChildCount()); i++ {
			child := node.Child(i)
			if child.Type() != "," && child.Type() != "{" && child.Type() != "}" {
				extractRustUseItems(child, source, code, entities, importNode)
			}
		}
	case "scoped_identifier":
		name := getLastSegment(node, code)
		*entities = append(*entities, createImportEntity(importNode, name, source, code))
	case "identifier":
		name := string(code[node.StartByte():node.EndByte()])
		*entities = append(*entities, createImportEntity(importNode, name, source, code))
	case "use_as_clause":
		if alias := node.ChildByFieldName("alias"); alias != nil {
			name := string(code[alias.StartByte():alias.EndByte()])
			*entities = append(*entities, createImportEntity(importNode, name, source, code))
		}
	case "use_wildcard":
		*entities = append(*entities, createImportEntity(importNode, "*", source, code))
	}
}

func getLastSegment(node *sitter.Node, code []byte) string {
	text := string(code[node.StartByte():node.EndByte()])
	parts := strings.Split(text, "::")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return text
}

func extractJavaImportSymbols(node *sitter.Node, source string, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)

	name := source
	parts := strings.Split(source, ".")
	if len(parts) > 0 {
		name = parts[len(parts)-1]
	}

	entities = append(entities, createImportEntity(node, name, source, code))
	return entities
}

func createImportEntity(node *sitter.Node, name, source string, code []byte) *ExtractedEntity {
	signature := string(code[node.StartByte():node.EndByte()])
	signature = cleanSignature(signature)

	var sourcePtr *string
	if source != "" {
		sourcePtr = &source
	}

	return &ExtractedEntity{
		Type:      EntityTypeImport,
		Name:      name,
		Signature: signature,
		Docstring: nil,
		ByteRange: ByteRange{
			Start: int(node.StartByte()),
			End:   int(node.EndByte()),
		},
		LineRange: LineRange{
			Start: int(node.StartPoint().Row),
			End:   int(node.EndPoint().Row),
		},
		Parent: nil,
		Node:   node,
		Source: sourcePtr,
	}
}
