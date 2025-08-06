package crawler

import (
	"bytes"
	"fmt"
	"net/url"
	"slices"

	"golang.org/x/net/html"
)

type ParseResult struct {
	Links  []string
	Assets []string
}

func (p *ParseResult) String() string {
	return fmt.Sprintf("Parsed links: %d, assets: %d", len(p.Links), len(p.Assets))
}

type Parser struct {
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) Parse(page *Page) (*ParseResult, error) {
	node, err := html.Parse(bytes.NewReader(page.Content))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var links []string
	for _, v := range extractNodesAttr("a", "href", node) {
		if u, ok := normalizeHref(v, page.URL); ok {
			links = append(links, u)
		}
	}

	// uniq
	slices.Sort(links)
	links = slices.Compact(links)

	var images []string
	for _, v := range extractNodesAttr("img", "src", node) {
		images = append(images, resolveAbsoluteUrl(v, page.URL))
	}

	return &ParseResult{
		Links:  links,
		Assets: append([]string{}, images...),
	}, nil
}

func extractNodesAttr(tag string, attr string, node *html.Node) []string {
	var res []string

	if node.Type == html.ElementNode && node.Data == tag {
		if val, ok := readNodeAttrValue(attr, node); ok {
			res = append(res, val)
		}
	}

	// recursive walk
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		res = append(res, extractNodesAttr(tag, attr, c)...)
	}

	return res
}

func readNodeAttrValue(attrName string, node *html.Node) (string, bool) {
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val, true
		}
	}

	return "", false
}

func resolveAbsoluteUrl(localUrl string, currentUrl string) string {
	base, err := url.Parse(currentUrl)
	if err != nil {
		return currentUrl + localUrl
	}

	ref, err := url.Parse(localUrl)
	if err != nil {
		return currentUrl + localUrl
	}

	return base.ResolveReference(ref).String()
}

func normalizeHref(href string, currentURL string) (string, bool) {
	switch {
	case href == "":
	case href == "/":
	case href == "#":
	case href[0] == '#':
		// ignoring
		return "", false
	default:
		return resolveAbsoluteUrl(href, currentURL), true
	}

	return "", false
}
