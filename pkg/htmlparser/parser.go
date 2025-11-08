package htmlparser

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"slices"
)

type ResourceNode struct {
	Node *html.Node
	Tag  string
	Src  string
}

// ParseResources парсит html и возвращает данные как есть.
func ParseResources(pageContent []byte) ([]*ResourceNode, error) {
	rootNode, err := html.Parse(bytes.NewBuffer(pageContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	return collect(rootNode, []string{"a", "link", "script", "img"}, func(node *html.Node) (*ResourceNode, bool) {
		tag := node.Data

		var src string
		var ok bool

		switch tag {
		case "script", "img":
			src, ok = readAttrValue(node, "src")
		case "link":
			typeAttr, _ := readAttrValue(node, "type")
			relAttr, _ := readAttrValue(node, "rel")
			if typeAttr == "text/css" || relAttr == "stylesheet" {
				src, ok = readAttrValue(node, "href")
			}
		case "a":
			src, ok = readAttrValue(node, "href")
		}

		if !ok {
			return nil, false
		}

		return &ResourceNode{
			Node: node,
			Tag:  tag,
			Src:  src,
		}, true
	}), nil
}

// collect обходит все узлы и собирает рекурсивно ResourceNode
func collect(node *html.Node, tags []string, match func(*html.Node) (*ResourceNode, bool)) []*ResourceNode {
	var res []*ResourceNode

	if node.Type == html.ElementNode && slices.Contains(tags, node.Data) {
		if val, ok := match(node); ok {
			res = append(res, val)
		}
	}

	// recursive walk
	for nextNode := node.FirstChild; nextNode != nil; nextNode = nextNode.NextSibling {
		res = append(res, collect(nextNode, tags, match)...)
	}

	return res
}

func readAttrValue(node *html.Node, attrName string) (string, bool) {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val, true
		}
	}

	return "", false
}
