package codechunk

import (
	"testing"
)

func TestRangeContains(t *testing.T) {
	tests := []struct {
		outer    ByteRange
		inner    ByteRange
		expected bool
	}{
		{ByteRange{0, 100}, ByteRange{10, 50}, true},
		{ByteRange{0, 100}, ByteRange{0, 100}, true},
		{ByteRange{10, 50}, ByteRange{0, 100}, false},
		{ByteRange{0, 50}, ByteRange{25, 75}, false},
		{ByteRange{0, 100}, ByteRange{0, 50}, true},
		{ByteRange{50, 100}, ByteRange{0, 50}, false},
	}

	for _, tt := range tests {
		result := rangeContains(tt.outer, tt.inner)
		if result != tt.expected {
			t.Errorf("rangeContains(%v, %v) = %v, want %v", tt.outer, tt.inner, result, tt.expected)
		}
	}
}

func TestCreateScopeNode(t *testing.T) {
	entity := &ExtractedEntity{
		Name: "TestFunc",
		Type: EntityTypeFunction,
	}

	// Test with nil parent
	node := createScopeNode(entity, nil)
	if node.Entity != entity {
		t.Error("Entity not set correctly")
	}
	if node.Parent != nil {
		t.Error("Parent should be nil")
	}
	if node.Children == nil {
		t.Error("Children should be initialized")
	}

	// Test with parent
	parent := &ScopeNode{Entity: &ExtractedEntity{Name: "Parent"}}
	childNode := createScopeNode(entity, parent)
	if childNode.Parent != parent {
		t.Error("Parent not set correctly")
	}
}

func TestBuildScopeTree(t *testing.T) {
	entities := []*ExtractedEntity{
		{
			Name:      "func1",
			Type:      EntityTypeFunction,
			ByteRange: ByteRange{0, 50},
		},
		{
			Name:      "func2",
			Type:      EntityTypeFunction,
			ByteRange: ByteRange{60, 100},
		},
		{
			Name:      "importA",
			Type:      EntityTypeImport,
			ByteRange: ByteRange{0, 20},
		},
	}

	tree := buildScopeTree(entities)

	if tree == nil {
		t.Fatal("Tree should not be nil")
	}

	if len(tree.Root) != 2 {
		t.Errorf("Expected 2 root nodes, got %d", len(tree.Root))
	}

	if len(tree.Imports) != 1 {
		t.Errorf("Expected 1 import, got %d", len(tree.Imports))
	}

	if len(tree.AllEntities) != 3 {
		t.Errorf("Expected 3 entities, got %d", len(tree.AllEntities))
	}
}

func TestBuildScopeTreeNested(t *testing.T) {
	entities := []*ExtractedEntity{
		{
			Name:      "outerClass",
			Type:      EntityTypeClass,
			ByteRange: ByteRange{0, 200},
		},
		{
			Name:      "innerMethod",
			Type:      EntityTypeMethod,
			ByteRange: ByteRange{50, 150},
		},
	}

	tree := buildScopeTree(entities)

	if len(tree.Root) != 1 {
		t.Errorf("Expected 1 root node, got %d", len(tree.Root))
	}

	rootNode := tree.Root[0]
	if len(rootNode.Children) != 1 {
		t.Errorf("Expected 1 child node, got %d", len(rootNode.Children))
	}

	childNode := rootNode.Children[0]
	if childNode.Entity.Name != "innerMethod" {
		t.Errorf("Expected child name 'innerMethod', got '%s'", childNode.Entity.Name)
	}

	if childNode.Parent != rootNode {
		t.Error("Child's parent should be the root node")
	}
}

func TestBuildScopeTreeEmpty(t *testing.T) {
	tree := buildScopeTree([]*ExtractedEntity{})

	if tree == nil {
		t.Fatal("Tree should not be nil")
	}

	if len(tree.Root) != 0 {
		t.Errorf("Expected 0 root nodes, got %d", len(tree.Root))
	}
}

func TestFindScopeAtOffset(t *testing.T) {
	entities := []*ExtractedEntity{
		{
			Name:      "func1",
			Type:      EntityTypeFunction,
			ByteRange: ByteRange{0, 50},
		},
		{
			Name:      "func2",
			Type:      EntityTypeFunction,
			ByteRange: ByteRange{60, 100},
		},
	}

	tree := buildScopeTree(entities)

	// Find scope at offset within func1
	node := findScopeAtOffset(tree, 25)
	if node == nil {
		t.Fatal("Expected to find scope node")
	}
	if node.Entity.Name != "func1" {
		t.Errorf("Expected 'func1', got '%s'", node.Entity.Name)
	}

	// Find scope at offset within func2
	node = findScopeAtOffset(tree, 80)
	if node == nil {
		t.Fatal("Expected to find scope node")
	}
	if node.Entity.Name != "func2" {
		t.Errorf("Expected 'func2', got '%s'", node.Entity.Name)
	}

	// Find scope at offset outside any function
	node = findScopeAtOffset(tree, 55)
	if node != nil {
		t.Error("Expected nil for offset between functions")
	}
}

func TestFindScopeAtOffsetNested(t *testing.T) {
	entities := []*ExtractedEntity{
		{
			Name:      "outerClass",
			Type:      EntityTypeClass,
			ByteRange: ByteRange{0, 200},
		},
		{
			Name:      "innerMethod",
			Type:      EntityTypeMethod,
			ByteRange: ByteRange{50, 150},
		},
	}

	tree := buildScopeTree(entities)

	// Find scope at offset within inner method (should return innermost)
	node := findScopeAtOffset(tree, 100)
	if node == nil {
		t.Fatal("Expected to find scope node")
	}
	if node.Entity.Name != "innerMethod" {
		t.Errorf("Expected 'innerMethod', got '%s'", node.Entity.Name)
	}

	// Find scope at offset within class but outside method
	node = findScopeAtOffset(tree, 25)
	if node == nil {
		t.Fatal("Expected to find scope node")
	}
	if node.Entity.Name != "outerClass" {
		t.Errorf("Expected 'outerClass', got '%s'", node.Entity.Name)
	}
}

func TestGetAncestorChain(t *testing.T) {
	grandparent := &ScopeNode{Entity: &ExtractedEntity{Name: "grandparent"}}
	parent := &ScopeNode{Entity: &ExtractedEntity{Name: "parent"}, Parent: grandparent}
	child := &ScopeNode{Entity: &ExtractedEntity{Name: "child"}, Parent: parent}

	ancestors := getAncestorChain(child)

	if len(ancestors) != 2 {
		t.Errorf("Expected 2 ancestors, got %d", len(ancestors))
	}

	if ancestors[0].Entity.Name != "parent" {
		t.Errorf("First ancestor should be parent, got '%s'", ancestors[0].Entity.Name)
	}

	if ancestors[1].Entity.Name != "grandparent" {
		t.Errorf("Second ancestor should be grandparent, got '%s'", ancestors[1].Entity.Name)
	}
}

func TestGetAncestorChainNoParent(t *testing.T) {
	node := &ScopeNode{Entity: &ExtractedEntity{Name: "root"}, Parent: nil}

	ancestors := getAncestorChain(node)

	if len(ancestors) != 0 {
		t.Errorf("Expected 0 ancestors, got %d", len(ancestors))
	}
}

func TestFlattenScopeTree(t *testing.T) {
	entities := []*ExtractedEntity{
		{
			Name:      "class1",
			Type:      EntityTypeClass,
			ByteRange: ByteRange{0, 100},
		},
		{
			Name:      "method1",
			Type:      EntityTypeMethod,
			ByteRange: ByteRange{10, 50},
		},
		{
			Name:      "class2",
			Type:      EntityTypeClass,
			ByteRange: ByteRange{110, 200},
		},
	}

	tree := buildScopeTree(entities)
	flat := flattenScopeTree(tree)

	// Should have class1, method1 (child of class1), and class2
	if len(flat) != 3 {
		t.Errorf("Expected 3 flattened nodes, got %d", len(flat))
	}
}

func TestFlattenScopeTreeEmpty(t *testing.T) {
	tree := buildScopeTree([]*ExtractedEntity{})
	flat := flattenScopeTree(tree)

	if len(flat) != 0 {
		t.Errorf("Expected 0 flattened nodes, got %d", len(flat))
	}
}

func TestSortByByteRange(t *testing.T) {
	entities := []*ExtractedEntity{
		{Name: "c", ByteRange: ByteRange{100, 150}},
		{Name: "a", ByteRange: ByteRange{0, 50}},
		{Name: "b", ByteRange: ByteRange{50, 100}},
	}

	sortByByteRange(entities)

	expectedOrder := []string{"a", "b", "c"}
	for i, entity := range entities {
		if entity.Name != expectedOrder[i] {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expectedOrder[i], entity.Name)
		}
	}
}

func TestSortByByteRangeAlreadySorted(t *testing.T) {
	entities := []*ExtractedEntity{
		{Name: "a", ByteRange: ByteRange{0, 50}},
		{Name: "b", ByteRange: ByteRange{50, 100}},
		{Name: "c", ByteRange: ByteRange{100, 150}},
	}

	sortByByteRange(entities)

	expectedOrder := []string{"a", "b", "c"}
	for i, entity := range entities {
		if entity.Name != expectedOrder[i] {
			t.Errorf("Position %d: expected '%s', got '%s'", i, expectedOrder[i], entity.Name)
		}
	}
}

func TestFindParentNodeNotFound(t *testing.T) {
	// Entity outside all ranges
	entity := &ExtractedEntity{
		Name:      "outside",
		ByteRange: ByteRange{1000, 1100},
	}

	roots := []*ScopeNode{
		{
			Entity: &ExtractedEntity{
				Name:      "root",
				ByteRange: ByteRange{0, 100},
			},
			Children: []*ScopeNode{},
		},
	}

	parent := findParentNode(roots, entity)
	if parent != nil {
		t.Error("Expected nil when entity is outside all ranges")
	}
}
