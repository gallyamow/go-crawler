package crawler

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
)

type ParseResult struct {
	Links     []string
	ImageUrls []string
}

func (p *ParseResult) String() string {
	return fmt.Sprintf("Parsed links: %d, images: %d bytes", len(p.Links), len(p.ImageUrls))
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

	links := extractLinks(node)
	return &ParseResult{
		Links: links,
	}, nil
}

func extractLinks(node *html.Node) []string {
	var links []string

	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" && attr.Val != "" {
				links = append(links, attr.Val)
			}
		}
	}

	// recursive walk
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		links = append(links, extractLinks(c)...)
	}

	return links
}
