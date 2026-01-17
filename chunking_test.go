package codechunk

import (
	"testing"

	sitter "github.com/smacker/go-tree-sitter"
)

func TestCountNws(t *testing.T) {
	tests := []struct {
		text     string
		expected int
	}{
		{"hello", 5},
		{"hello world", 10},
		{"  hello  ", 5},
		{"\t\n  test\t\n", 4},
		{"", 0},
		{"   ", 0},
		{"\t\n\r", 0},
		{"abc123!@#", 9},
	}

	for _, tt := range tests {
		result := countNws(tt.text)
		if result != tt.expected {
			t.Errorf("countNws(%q) = %d, want %d", tt.text, result, tt.expected)
		}
	}
}

func TestIsWhitespace(t *testing.T) {
	// Test whitespace characters
	whitespace := []byte{' ', '\t', '\n', '\r', 0}
	for _, c := range whitespace {
		if !isWhitespace(c) {
			t.Errorf("isWhitespace(%d) = false, want true", c)
		}
	}

	// Test non-whitespace characters
	nonWhitespace := []byte{'a', 'Z', '0', '!', '@', '#'}
	for _, c := range nonWhitespace {
		if isWhitespace(c) {
			t.Errorf("isWhitespace(%d) = true, want false", c)
		}
	}
}

func TestPreprocessNwsCumsum(t *testing.T) {
	tests := []struct {
		code     string
		expected []uint32
	}{
		{"abc", []uint32{0, 1, 2, 3}},
		{"a b", []uint32{0, 1, 1, 2}},
		{"  ", []uint32{0, 0, 0}},
		{"", []uint32{0}},
		{"a\nb", []uint32{0, 1, 1, 2}},
	}

	for _, tt := range tests {
		result := preprocessNwsCumsum([]byte(tt.code))
		if len(result) != len(tt.expected) {
			t.Errorf("preprocessNwsCumsum(%q) length = %d, want %d", tt.code, len(result), len(tt.expected))
			continue
		}
		for i, v := range tt.expected {
			if result[i] != v {
				t.Errorf("preprocessNwsCumsum(%q)[%d] = %d, want %d", tt.code, i, result[i], v)
			}
		}
	}
}

func TestGetNwsCountFromCumsum(t *testing.T) {
	code := []byte("hello world")
	cumsum := preprocessNwsCumsum(code)

	tests := []struct {
		start    int
		end      int
		expected int
	}{
		{0, 5, 5},   // "hello"
		{6, 11, 5},  // "world"
		{0, 11, 10}, // "hello world" (without space)
		{0, 0, 0},   // empty range
		{-1, 5, 5},  // negative start clamped to 0
		{0, 100, 10}, // end beyond length clamped
	}

	for _, tt := range tests {
		result := getNwsCountFromCumsum(cumsum, tt.start, tt.end)
		if result != tt.expected {
			t.Errorf("getNwsCountFromCumsum(cumsum, %d, %d) = %d, want %d", tt.start, tt.end, result, tt.expected)
		}
	}
}

func TestCountNewlines(t *testing.T) {
	tests := []struct {
		code     string
		start    int
		end      int
		expected int
	}{
		{"hello\nworld", 0, 11, 1},
		{"a\nb\nc", 0, 5, 2},
		{"no newlines", 0, 11, 0},
		{"\n\n\n", 0, 3, 3},
		{"test", 0, 100, 0}, // end beyond length
	}

	for _, tt := range tests {
		result := countNewlines([]byte(tt.code), tt.start, tt.end)
		if result != tt.expected {
			t.Errorf("countNewlines(%q, %d, %d) = %d, want %d", tt.code, tt.start, tt.end, result, tt.expected)
		}
	}
}

func TestCountLinesUpTo(t *testing.T) {
	tests := []struct {
		code     string
		offset   int
		expected int
	}{
		{"hello\nworld", 5, 0},
		{"hello\nworld", 6, 1},
		{"a\nb\nc", 3, 1}, // offset 3 is after first \n at position 1
		{"test", 4, 0},
		{"test", 100, 0}, // offset beyond length
	}

	for _, tt := range tests {
		result := countLinesUpTo([]byte(tt.code), tt.offset)
		if result != tt.expected {
			t.Errorf("countLinesUpTo(%q, %d) = %d, want %d", tt.code, tt.offset, result, tt.expected)
		}
	}
}

func TestMergeAdjacentWindows(t *testing.T) {
	// Test empty input
	result := mergeAdjacentWindows([]*ASTWindow{}, 100)
	if len(result) != 0 {
		t.Error("mergeAdjacentWindows([]) should return empty slice")
	}

	// Test single window
	singleWindow := []*ASTWindow{
		{Size: 50},
	}
	result = mergeAdjacentWindows(singleWindow, 100)
	if len(result) != 1 {
		t.Errorf("mergeAdjacentWindows single window should return 1, got %d", len(result))
	}

	// Test mergeable windows
	windows := []*ASTWindow{
		{Size: 30, Nodes: nil, Ancestors: nil},
		{Size: 40, Nodes: nil, Ancestors: nil},
		{Size: 20, Nodes: nil, Ancestors: nil},
	}
	result = mergeAdjacentWindows(windows, 100)
	if len(result) != 1 {
		t.Errorf("mergeAdjacentWindows should merge 3 small windows into 1, got %d", len(result))
	}

	// Test non-mergeable windows
	largeWindows := []*ASTWindow{
		{Size: 60, Nodes: nil, Ancestors: nil},
		{Size: 60, Nodes: nil, Ancestors: nil},
	}
	result = mergeAdjacentWindows(largeWindows, 100)
	if len(result) != 2 {
		t.Errorf("mergeAdjacentWindows should not merge large windows, got %d", len(result))
	}
}

func TestRebuildText(t *testing.T) {
	// Test empty window
	emptyWindow := &ASTWindow{
		Nodes: []*sitter.Node{},
	}
	result := rebuildText(emptyWindow, []byte("code"))
	if result.text != "" {
		t.Error("rebuildText empty window should return empty text")
	}
}

func TestIsLeafNode(t *testing.T) {
	// Parse a simple code to get nodes for testing
	code := `func main() {}`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	root := parseResult.Tree.RootNode()

	// Root is not a leaf
	if isLeafNode(root) {
		t.Error("Root should not be a leaf node")
	}

	// Find a leaf node (identifier)
	var leafNode *sitter.Node
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.ChildCount() == 0 {
			leafNode = child
			break
		}
		// Check grandchildren
		for j := 0; j < int(child.ChildCount()); j++ {
			grandchild := child.Child(j)
			if grandchild.ChildCount() == 0 {
				leafNode = grandchild
				break
			}
		}
		if leafNode != nil {
			break
		}
	}

	if leafNode != nil && !isLeafNode(leafNode) {
		t.Error("Leaf node should be identified as leaf")
	}
}

func TestGetAncestorsForNodes(t *testing.T) {
	// Test empty list
	result := getAncestorsForNodes(nil)
	if result != nil {
		t.Error("getAncestorsForNodes(nil) should return nil")
	}

	result = getAncestorsForNodes([]*sitter.Node{})
	if result != nil {
		t.Error("getAncestorsForNodes([]) should return nil")
	}
}

func TestGetNodeChildren(t *testing.T) {
	// Test with nil
	result := getNodeChildren(nil)
	if result != nil {
		t.Error("getNodeChildren(nil) should return nil")
	}

	// Test with invalid type
	result = getNodeChildren("not a node")
	if result != nil {
		t.Error("getNodeChildren(invalid) should return nil")
	}

	// Test with actual node
	code := `func main() {}`
	parseResult, err := parseString(code, LanguageGo)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	root := parseResult.Tree.RootNode()
	children := getNodeChildren(root)
	if len(children) == 0 {
		t.Error("Root node should have children")
	}
}

