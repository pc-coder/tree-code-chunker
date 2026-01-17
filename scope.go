package codechunk

// rangeContains checks if outer range fully contains inner range
func rangeContains(outer, inner ByteRange) bool {
	return outer.Start <= inner.Start && inner.End <= outer.End
}

// createScopeNode creates a new scope node from an entity
func createScopeNode(entity *ExtractedEntity, parent *ScopeNode) *ScopeNode {
	return &ScopeNode{
		Entity:   entity,
		Children: make([]*ScopeNode, 0),
		Parent:   parent,
	}
}

// findParentNode finds the deepest parent node whose range contains the entity's range
func findParentNode(roots []*ScopeNode, entity *ExtractedEntity) *ScopeNode {
	for _, root := range roots {
		if found := findInNode(root, entity); found != nil {
			return found
		}
	}
	return nil
}

func findInNode(node *ScopeNode, entity *ExtractedEntity) *ScopeNode {
	if !rangeContains(node.Entity.ByteRange, entity.ByteRange) {
		return nil
	}

	for _, child := range node.Children {
		if deeperMatch := findInNode(child, entity); deeperMatch != nil {
			return deeperMatch
		}
	}

	return node
}

// buildScopeTree builds a scope tree from extracted entities
func buildScopeTree(entities []*ExtractedEntity) *ScopeTree {
	imports := make([]*ExtractedEntity, 0)
	exports := make([]*ExtractedEntity, 0)
	scopeEntities := make([]*ExtractedEntity, 0)

	for _, entity := range entities {
		switch entity.Type {
		case EntityTypeImport:
			imports = append(imports, entity)
		case EntityTypeExport:
			exports = append(exports, entity)
		default:
			scopeEntities = append(scopeEntities, entity)
		}
	}

	// Sort by byte range start
	sortByByteRange(scopeEntities)

	root := make([]*ScopeNode, 0)

	for _, entity := range scopeEntities {
		parent := findParentNode(root, entity)
		node := createScopeNode(entity, parent)

		if parent != nil {
			parent.Children = append(parent.Children, node)
		} else {
			root = append(root, node)
		}
	}

	return &ScopeTree{
		Root:        root,
		Imports:     imports,
		Exports:     exports,
		AllEntities: entities,
	}
}

// sortByByteRange sorts entities by byte range start
func sortByByteRange(entities []*ExtractedEntity) {
	for i := 1; i < len(entities); i++ {
		key := entities[i]
		j := i - 1
		for j >= 0 && entities[j].ByteRange.Start > key.ByteRange.Start {
			entities[j+1] = entities[j]
			j--
		}
		entities[j+1] = key
	}
}

// findScopeAtOffset finds the scope node that contains a given byte offset
func findScopeAtOffset(tree *ScopeTree, offset int) *ScopeNode {
	for _, root := range tree.Root {
		if found := findScopeInNode(root, offset); found != nil {
			return found
		}
	}
	return nil
}

func findScopeInNode(node *ScopeNode, offset int) *ScopeNode {
	byteRange := node.Entity.ByteRange

	if offset < byteRange.Start || offset >= byteRange.End {
		return nil
	}

	for _, child := range node.Children {
		if deeperMatch := findScopeInNode(child, offset); deeperMatch != nil {
			return deeperMatch
		}
	}

	return node
}

// getAncestorChain gets the ancestor chain for a scope node
func getAncestorChain(node *ScopeNode) []*ScopeNode {
	ancestors := make([]*ScopeNode, 0)
	current := node.Parent
	for current != nil {
		ancestors = append(ancestors, current)
		current = current.Parent
	}
	return ancestors
}

// flattenScopeTree flattens a scope tree into a list of all scope nodes
func flattenScopeTree(tree *ScopeTree) []*ScopeNode {
	result := make([]*ScopeNode, 0)
	for _, root := range tree.Root {
		visitNode(root, &result)
	}
	return result
}

func visitNode(node *ScopeNode, result *[]*ScopeNode) {
	*result = append(*result, node)
	for _, child := range node.Children {
		visitNode(child, result)
	}
}
