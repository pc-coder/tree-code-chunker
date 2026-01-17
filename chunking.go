package codechunk

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// nwsCumsum is a cumulative sum array for O(1) NWS range queries
type nwsCumsum []uint32

// countNws counts non-whitespace characters in a string
func countNws(text string) int {
	count := 0
	for i := 0; i < len(text); i++ {
		if !isWhitespace(text[i]) {
			count++
		}
	}
	return count
}

func isWhitespace(c byte) bool {
	return c <= 32
}

// preprocessNwsCumsum preprocesses code for O(1) NWS range queries
func preprocessNwsCumsum(code []byte) nwsCumsum {
	cumsum := make(nwsCumsum, len(code)+1)
	count := uint32(0)
	for i := 0; i < len(code); i++ {
		if !isWhitespace(code[i]) {
			count++
		}
		cumsum[i+1] = count
	}
	return cumsum
}

// getNwsCountFromCumsum gets NWS count for a range (O(1))
func getNwsCountFromCumsum(cumsum nwsCumsum, start, end int) int {
	if end > len(cumsum)-1 {
		end = len(cumsum) - 1
	}
	if start < 0 {
		start = 0
	}
	return int(cumsum[end] - cumsum[start])
}

// getNwsCountForNode gets NWS count for a node (O(1))
func getNwsCountForNode(node *sitter.Node, cumsum nwsCumsum) int {
	return getNwsCountFromCumsum(cumsum, int(node.StartByte()), int(node.EndByte()))
}

// isLeafNode checks if a node has no children
func isLeafNode(node *sitter.Node) bool {
	return node.ChildCount() == 0
}

// getAncestorsForNodes gets ancestor nodes for the first node in a list
func getAncestorsForNodes(nodes []*sitter.Node) []*sitter.Node {
	if len(nodes) == 0 {
		return nil
	}

	ancestors := make([]*sitter.Node, 0)
	current := nodes[0].Parent()
	for current != nil {
		ancestors = append(ancestors, current)
		current = current.Parent()
	}
	return ancestors
}

// greedyAssignWindows assigns nodes to windows using a greedy algorithm
func greedyAssignWindows(nodes []*sitter.Node, code []byte, cumsum nwsCumsum, maxSize int) []*ASTWindow {
	windows := make([]*ASTWindow, 0)
	currentWindow := &ASTWindow{
		Nodes:     make([]*sitter.Node, 0),
		Ancestors: make([]*sitter.Node, 0),
		Size:      0,
	}

	for _, node := range nodes {
		nodeSize := getNwsCountForNode(node, cumsum)

		if currentWindow.Size+nodeSize <= maxSize {
			currentWindow.Nodes = append(currentWindow.Nodes, node)
			currentWindow.Size += nodeSize
		} else if nodeSize > maxSize {
			if len(currentWindow.Nodes) > 0 {
				currentWindow.Ancestors = getAncestorsForNodes(currentWindow.Nodes)
				windows = append(windows, currentWindow)
				currentWindow = &ASTWindow{
					Nodes:     make([]*sitter.Node, 0),
					Ancestors: make([]*sitter.Node, 0),
					Size:      0,
				}
			}

			if !isLeafNode(node) {
				children := make([]*sitter.Node, 0, node.ChildCount())
				for i := 0; i < int(node.ChildCount()); i++ {
					if child := node.Child(i); child != nil {
						children = append(children, child)
					}
				}
				childWindows := greedyAssignWindows(children, code, cumsum, maxSize)
				windows = append(windows, childWindows...)
			} else {
				leafWindows := splitOversizedLeafByLines(node, code, maxSize)
				windows = append(windows, leafWindows...)
			}
		} else {
			if len(currentWindow.Nodes) > 0 {
				currentWindow.Ancestors = getAncestorsForNodes(currentWindow.Nodes)
				windows = append(windows, currentWindow)
			}
			currentWindow = &ASTWindow{
				Nodes:     []*sitter.Node{node},
				Ancestors: make([]*sitter.Node, 0),
				Size:      nodeSize,
			}
		}
	}

	if len(currentWindow.Nodes) > 0 {
		currentWindow.Ancestors = getAncestorsForNodes(currentWindow.Nodes)
		windows = append(windows, currentWindow)
	}

	return windows
}

// splitOversizedLeafByLines splits an oversized leaf node at line boundaries
func splitOversizedLeafByLines(node *sitter.Node, code []byte, maxSize int) []*ASTWindow {
	windows := make([]*ASTWindow, 0)

	text := string(code[node.StartByte():node.EndByte()])
	lines := strings.Split(text, "\n")

	var currentChunk strings.Builder
	currentSize := 0
	startByte := int(node.StartByte())
	chunkStartOffset := 0

	for i, line := range lines {
		lineNws := countNws(line)
		lineWithNewline := line
		if i < len(lines)-1 {
			lineWithNewline += "\n"
		}

		if currentSize+lineNws <= maxSize {
			currentChunk.WriteString(lineWithNewline)
			currentSize += lineNws
		} else {
			if currentChunk.Len() > 0 {
				startLine := countNewlines(code, 0, startByte+chunkStartOffset)
				endLine := countNewlines(code, 0, startByte+chunkStartOffset+currentChunk.Len())

				windows = append(windows, &ASTWindow{
					Nodes:         []*sitter.Node{node},
					Ancestors:     getAncestorsForNodes([]*sitter.Node{node}),
					Size:          currentSize,
					IsPartialNode: true,
					LineRanges: []LineRange{
						{Start: startLine, End: endLine},
					},
				})
			}

			chunkStartOffset += currentChunk.Len()
			currentChunk.Reset()
			currentChunk.WriteString(lineWithNewline)
			currentSize = lineNws
		}
	}

	if currentChunk.Len() > 0 {
		startLine := countNewlines(code, 0, startByte+chunkStartOffset)
		endLine := countNewlines(code, 0, startByte+chunkStartOffset+currentChunk.Len())

		windows = append(windows, &ASTWindow{
			Nodes:         []*sitter.Node{node},
			Ancestors:     getAncestorsForNodes([]*sitter.Node{node}),
			Size:          currentSize,
			IsPartialNode: true,
			LineRanges: []LineRange{
				{Start: startLine, End: endLine},
			},
		})
	}

	return windows
}

// countNewlines counts newlines in code from start to end offset
func countNewlines(code []byte, start, end int) int {
	if end > len(code) {
		end = len(code)
	}
	count := 0
	for i := start; i < end; i++ {
		if code[i] == '\n' {
			count++
		}
	}
	return count
}

// mergeAdjacentWindows merges adjacent windows that fit within maxSize
func mergeAdjacentWindows(windows []*ASTWindow, maxSize int) []*ASTWindow {
	if len(windows) == 0 {
		return windows
	}

	merged := make([]*ASTWindow, 0)
	current := windows[0]

	for i := 1; i < len(windows); i++ {
		next := windows[i]

		if current.Size+next.Size <= maxSize {
			current = &ASTWindow{
				Nodes:         append(current.Nodes, next.Nodes...),
				Ancestors:     current.Ancestors,
				Size:          current.Size + next.Size,
				IsPartialNode: current.IsPartialNode || next.IsPartialNode,
				LineRanges:    append(current.LineRanges, next.LineRanges...),
			}
		} else {
			merged = append(merged, current)
			current = next
		}
	}

	merged = append(merged, current)

	return merged
}

// rebuiltText represents text rebuilt from an AST window
type rebuiltText struct {
	text      string
	byteRange ByteRange
	lineRange LineRange
}

// rebuildText rebuilds text from an AST window
func rebuildText(window *ASTWindow, code []byte) *rebuiltText {
	if len(window.Nodes) == 0 {
		return &rebuiltText{
			text:      "",
			byteRange: ByteRange{Start: 0, End: 0},
			lineRange: LineRange{Start: 0, End: 0},
		}
	}

	startByte := int(window.Nodes[0].StartByte())
	endByte := int(window.Nodes[0].EndByte())

	for _, node := range window.Nodes[1:] {
		if int(node.StartByte()) < startByte {
			startByte = int(node.StartByte())
		}
		if int(node.EndByte()) > endByte {
			endByte = int(node.EndByte())
		}
	}

	if endByte > len(code) {
		endByte = len(code)
	}
	if startByte < 0 {
		startByte = 0
	}

	text := string(code[startByte:endByte])

	// Trim trailing newlines to match TypeScript behavior
	// tree-sitter-wasm excludes trailing newlines from node ranges
	for len(text) > 0 && text[len(text)-1] == '\n' {
		text = text[:len(text)-1]
		endByte--
	}

	startLine := countLinesUpTo(code, startByte)
	endLine := countLinesUpTo(code, endByte)

	if len(window.LineRanges) > 0 {
		startLine = window.LineRanges[0].Start
		if len(window.LineRanges) > 0 {
			endLine = window.LineRanges[len(window.LineRanges)-1].End
		}
	}

	return &rebuiltText{
		text: text,
		byteRange: ByteRange{
			Start: startByte,
			End:   endByte,
		},
		lineRange: LineRange{
			Start: startLine,
			End:   endLine,
		},
	}
}

// countLinesUpTo counts newlines from 0 to offset
func countLinesUpTo(code []byte, offset int) int {
	if offset > len(code) {
		offset = len(code)
	}
	count := 0
	for i := 0; i < offset; i++ {
		if code[i] == '\n' {
			count++
		}
	}
	return count
}

// getNodeChildren gets children of a node
func getNodeChildren(node interface{}) []*sitter.Node {
	n, ok := node.(*sitter.Node)
	if !ok {
		return nil
	}

	children := make([]*sitter.Node, 0, n.ChildCount())
	for i := 0; i < int(n.ChildCount()); i++ {
		if child := n.Child(i); child != nil {
			children = append(children, child)
		}
	}
	return children
}
