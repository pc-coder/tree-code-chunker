package codechunk

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// EntityNodeTypes maps languages to node types that represent extractable entities
var EntityNodeTypes = map[Language][]string{
	LanguageTypeScript: {
		"function_declaration",
		"method_definition",
		"class_declaration",
		"abstract_class_declaration",
		"interface_declaration",
		"type_alias_declaration",
		"enum_declaration",
		"import_statement",
		"export_statement",
	},
	LanguageJavaScript: {
		"function_declaration",
		"generator_function_declaration",
		"method_definition",
		"class_declaration",
		"import_statement",
		"export_statement",
	},
	LanguagePython: {
		"function_definition",
		"class_definition",
		"import_statement",
		"import_from_statement",
	},
	LanguageRust: {
		"function_item",
		"impl_item",
		"struct_item",
		"enum_item",
		"trait_item",
		"type_item",
		"use_declaration",
	},
	LanguageGo: {
		"function_declaration",
		"method_declaration",
		"type_declaration",
		"import_declaration",
	},
	LanguageJava: {
		"method_declaration",
		"constructor_declaration",
		"class_declaration",
		"interface_declaration",
		"enum_declaration",
		"import_declaration",
	},
}

// NodeTypeToEntityType maps AST node types to entity types
var NodeTypeToEntityType = map[string]EntityType{
	// Functions
	"function_declaration":           EntityTypeFunction,
	"function_definition":            EntityTypeFunction,
	"function_item":                  EntityTypeFunction,
	"generator_function_declaration": EntityTypeFunction,
	"arrow_function":                 EntityTypeFunction,

	// Methods
	"method_definition":       EntityTypeMethod,
	"method_declaration":      EntityTypeMethod,
	"constructor_declaration": EntityTypeMethod,

	// Classes
	"class_declaration":          EntityTypeClass,
	"class_definition":           EntityTypeClass,
	"abstract_class_declaration": EntityTypeClass,
	"impl_item":                  EntityTypeClass,

	// Interfaces
	"interface_declaration": EntityTypeInterface,
	"trait_item":            EntityTypeInterface,

	// Types
	"type_alias_declaration": EntityTypeType,
	"type_item":              EntityTypeType,
	"type_declaration":       EntityTypeType,
	"struct_item":            EntityTypeType,

	// Enums
	"enum_declaration": EntityTypeEnum,
	"enum_item":        EntityTypeEnum,

	// Imports
	"import_statement":      EntityTypeImport,
	"import_declaration":    EntityTypeImport,
	"import_from_statement": EntityTypeImport,
	"use_declaration":       EntityTypeImport,

	// Exports
	"export_statement": EntityTypeExport,
}

// isEntityNodeType checks if a node type is an entity type for the given language
func isEntityNodeType(nodeType string, lang Language) bool {
	types, ok := EntityNodeTypes[lang]
	if !ok {
		return false
	}
	for _, t := range types {
		if t == nodeType {
			return true
		}
	}
	return false
}

// getEntityType gets EntityType from node type string
func getEntityType(nodeType string) (EntityType, bool) {
	entityType, ok := NodeTypeToEntityType[nodeType]
	return entityType, ok
}

// extractEntities extracts entities from an AST tree
func extractEntities(rootNode *sitter.Node, lang Language, code []byte) []*ExtractedEntity {
	entities := make([]*ExtractedEntity, 0)
	processedNodes := make(map[uintptr]bool)

	walkAndExtract(rootNode, lang, code, nil, &entities, processedNodes)

	return entities
}

// stackItem represents an item in the traversal stack
type stackItem struct {
	node       *sitter.Node
	parentName *string
}

// walkAndExtract walks the AST iteratively and extracts entities
func walkAndExtract(rootNode *sitter.Node, lang Language, code []byte, parentName *string, entities *[]*ExtractedEntity, processedNodes map[uintptr]bool) {
	stack := []stackItem{{node: rootNode, parentName: parentName}}

	for len(stack) > 0 {
		// Pop from stack
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		node := current.node
		if node == nil {
			continue
		}

		nodePtr := node.ID()

		// Check if this node is an entity type
		if isEntityNodeType(node.Type(), lang) {
			// Skip if already processed
			if processedNodes[nodePtr] {
				continue
			}
			processedNodes[nodePtr] = true

			entityType, ok := getEntityType(node.Type())
			if !ok {
				entityType = inferEntityType(node.Type())
				if entityType == "" {
					continue
				}
			}

			// For import statements, extract individual symbols
			if entityType == EntityTypeImport {
				importEntities := extractImportSymbols(node, lang, code)
				*entities = append(*entities, importEntities...)
			} else {
				// Extract name
				name := extractNameFromCode(node, code, lang)
				if name == "" {
					name = "<anonymous>"
				}

				// Extract signature
				signature := extractSignature(node, entityType, lang, code)
				if signature == "" {
					signature = name
				}

				// Extract docstring
				docstring := extractDocstring(node, lang, code)

				// Create entity
				entity := &ExtractedEntity{
					Type:      entityType,
					Name:      name,
					Signature: signature,
					Docstring: docstring,
					ByteRange: ByteRange{
						Start: int(node.StartByte()),
						End:   int(node.EndByte()),
					},
					LineRange: LineRange{
						Start: int(node.StartPoint().Row),
						End:   int(node.EndPoint().Row),
					},
					Parent: current.parentName,
					Node:   node,
				}

				*entities = append(*entities, entity)

				// For nested entities, use this entity's name as parent
				var newParentName *string
				if entityType == EntityTypeClass ||
					entityType == EntityTypeInterface ||
					entityType == EntityTypeFunction ||
					entityType == EntityTypeMethod {
					newParentName = &name
				} else {
					newParentName = current.parentName
				}

				// Add children to stack (in reverse order for correct DFS order)
				for i := int(node.ChildCount()) - 1; i >= 0; i-- {
					child := node.Child(i)
					if child != nil {
						stack = append(stack, stackItem{node: child, parentName: newParentName})
					}
				}
			}
		} else {
			// Not an entity node, but might contain entity nodes
			for i := int(node.ChildCount()) - 1; i >= 0; i-- {
				child := node.Child(i)
				if child != nil {
					stack = append(stack, stackItem{node: child, parentName: current.parentName})
				}
			}
		}
	}
}

// inferEntityType infers entity type from node type string
func inferEntityType(nodeType string) EntityType {
	lowerType := strings.ToLower(nodeType)

	switch {
	case strings.Contains(lowerType, "function") || strings.Contains(lowerType, "arrow"):
		return EntityTypeFunction
	case strings.Contains(lowerType, "method"):
		return EntityTypeMethod
	case strings.Contains(lowerType, "class"):
		return EntityTypeClass
	case strings.Contains(lowerType, "interface") || strings.Contains(lowerType, "trait"):
		return EntityTypeInterface
	case strings.Contains(lowerType, "type") || strings.Contains(lowerType, "struct"):
		return EntityTypeType
	case strings.Contains(lowerType, "enum"):
		return EntityTypeEnum
	case strings.Contains(lowerType, "import") || strings.Contains(lowerType, "use"):
		return EntityTypeImport
	case strings.Contains(lowerType, "export"):
		return EntityTypeExport
	default:
		return ""
	}
}

// nameNodeTypes are node types that represent identifiers/names
var nameNodeTypes = []string{
	"name",
	"identifier",
	"type_identifier",
	"property_identifier",
}

// extractNameFromCode extracts the name using the source code
func extractNameFromCode(node *sitter.Node, code []byte, lang Language) string {
	// Try to find a named child that is an identifier
	for _, nameType := range nameNodeTypes {
		if nameNode := node.ChildByFieldName(nameType); nameNode != nil {
			return string(code[nameNode.StartByte():nameNode.EndByte()])
		}
	}

	// Try to find any child with a name-like type
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		for _, nameType := range nameNodeTypes {
			if child.Type() == nameType {
				return string(code[child.StartByte():child.EndByte()])
			}
		}
	}

	// For some languages, try the first identifier child
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() == "identifier" || child.Type() == "type_identifier" {
			return string(code[child.StartByte():child.EndByte()])
		}
	}

	return ""
}
