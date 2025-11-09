package htmlparser

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"slices"
)

type HTMLResource struct {
	Node *html.Node
	Src  string
}

func (rn *HTMLResource) SetSrc(newSrc string) bool {
	switch rn.Tag() {
	case "script", "img":
		return setAttrValue(rn.Node, "newSrc", newSrc)
	case "link":
		return setAttrValue(rn.Node, "href", newSrc)
	case "a":
		return setAttrValue(rn.Node, "href", newSrc)
	}
	return false
}

func (rn *HTMLResource) Tag() string {
	return rn.Node.Data
}

// ParseHTMLResources парсит html и возвращает данные как есть.
func ParseHTMLResources(pageContent []byte) (*html.Node, []*HTMLResource, error) {
	rootNode, err := html.Parse(bytes.NewBuffer(pageContent))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse html: %w", err)
	}

	resources := collect(rootNode, []string{"a", "link", "script", "img"}, func(node *html.Node) (*HTMLResource, bool) {
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

		return &HTMLResource{
			Node: node,
			Src:  src,
		}, true
	})

	return rootNode, resources, nil
}

// collect обходит все узлы и собирает рекурсивно HTMLResource
func collect(node *html.Node, tags []string, match func(*html.Node) (*HTMLResource, bool)) []*HTMLResource {
	var res []*HTMLResource

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

func setAttrValue(node *html.Node, attrName string, attrValue string) bool {
	for i, attr := range node.Attr {
		if attr.Key == attrName {
			node.Attr[i].Val = attrValue
			return true
		}
	}
	return false
}
