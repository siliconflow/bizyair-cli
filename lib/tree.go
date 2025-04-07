package lib

import (
	"fmt"
	"sort"
	"strings"
)

// Node represents a node in the tree
type Node struct {
	Name     string
	Children map[string]*Node
}

// NewNode creates a new node
func NewNode(name string) *Node {
	return &Node{
		Name:     name,
		Children: make(map[string]*Node),
	}
}

// AddPath adds a path to the tree
func (n *Node) AddPath(path string) {
	parts := strings.Split(path, "/")
	current := n
	for _, part := range parts {
		if _, exists := current.Children[part]; !exists {
			current.Children[part] = NewNode(part)
		}
		current = current.Children[part]
	}
}

// PrintTree prints the tree in a tree-like format
func (n *Node) PrintTree(indent string) {
	keys := make([]string, 0, len(n.Children))
	for key := range n.Children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for i, key := range keys {
		if i == len(keys)-1 {
			fmt.Printf("%s└── %s\n", indent, key)
			node := n.Children[key]
			node.PrintTree(indent + "    ")
		} else {
			fmt.Printf("%s├── %s\n", indent, key)
			n.Children[key].PrintTree(indent + "│   ")
		}
	}
}
