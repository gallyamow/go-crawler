package htmlparser

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"slices"
)

type SrcNode struct {
	Node *html.Node
	Src  string
}

func (rn *SrcNode) Tag() string {
	return rn.Node.Data
}

// ParseResources парсит html и возвращает данные как есть.
func ParseResources(pageContent []byte) (*html.Node, []*SrcNode, error) {
	rootNode, err := html.Parse(bytes.NewBuffer(pageContent))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse html: %w", err)
	}

	resources := collect(rootNode, []string{"a", "link", "script", "img"}, func(node *html.Node) (*SrcNode, bool) {
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

		return &SrcNode{
			Node: node,
			Src:  src,
		}, true
	})

	return rootNode, resources, nil
}

// collect обходит все узлы и собирает рекурсивно SrcNode
func collect(node *html.Node, tags []string, match func(*html.Node) (*SrcNode, bool)) []*SrcNode {
	var res []*SrcNode

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
