package htmlparser

import (
	"bytes"
	"fmt"
	urllib "net/url"
	"slices"

	"golang.org/x/net/html"
)

type ParseResult struct {
	Links       []string
	Stylesheets []string
	Scripts     []string
	Images      []string
}

func Parse(rawPageURL string, pageContent []byte) (*ParseResult, error) {
	pageURL, err := urllib.Parse(rawPageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %w", err)
	}

	rootNode, err := html.Parse(bytes.NewBuffer(pageContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse html: %w", err)
	}

	return &ParseResult{
		Links: collectUniqueAttrValues(rootNode, "a", "href", nil, func(v string) (string, bool) {
			return normalizeURL(v, pageURL)
		}),
		Stylesheets: collectUniqueAttrValues(rootNode, "link", "href", func(node *html.Node) bool {
			typeAttr, _ := readAttrValue(node, "type")
			relAttr, _ := readAttrValue(node, "rel")

			return typeAttr == "text/css" || relAttr == "stylesheet"
		}, func(v string) (string, bool) {
			return normalizeURL(v, pageURL)
		}),
		Scripts: collectUniqueAttrValues(rootNode, "script", "src", nil, func(v string) (string, bool) {
			return normalizeURL(v, pageURL)
		}),
		Images: collectUniqueAttrValues(rootNode, "img", "src", nil, func(v string) (string, bool) {
			return normalizeURL(v, pageURL)
		}),
	}, nil
}

func collectUniqueAttrValues(rootNode *html.Node, tag string, attr string, nodeFilter func(*html.Node) bool, valueTransformer func(string) (string, bool)) []string {
	var vals []string

	for _, v := range collectAttrValues(rootNode, tag, attr, nodeFilter) {
		if t, ok := valueTransformer(v); ok {
			vals = append(vals, t)
		}
	}

	// unique
	slices.Sort(vals)
	vals = slices.Compact(vals)

	return vals
}

func collectAttrValues(node *html.Node, tag string, attr string, filter func(*html.Node) bool) []string {
	var res []string

	if node.Type == html.ElementNode && node.Data == tag {
		if filter == nil || filter(node) {
			if val, ok := readAttrValue(node, attr); ok {
				res = append(res, val)
			}
		}
	}

	// recursive walk
	for nextNode := node.FirstChild; nextNode != nil; nextNode = nextNode.NextSibling {
		res = append(res, collectAttrValues(nextNode, tag, attr, filter)...)
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

func normalizeURL(rawURL string, pageURL *urllib.URL) (string, bool) {
	var url *urllib.URL
	var ok bool

	if url, ok = parseURL(rawURL); ok {
		if url, ok = stripAnchor(url); ok {
			return resolveAbsoluteURL(url, pageURL).String(), true
		}
	}

	return "", false
}

func parseURL(rawURL string) (*urllib.URL, bool) {
	url, err := urllib.Parse(rawURL)
	if err != nil {
		return nil, false
	}
	return url, true
}

func resolveAbsoluteURL(localUrl *urllib.URL, pageUrl *urllib.URL) *urllib.URL {
	return pageUrl.ResolveReference(localUrl)
}

func stripAnchor(url *urllib.URL) (*urllib.URL, bool) {
	// drop anchor
	url.Fragment = ""

	return url, true
}
