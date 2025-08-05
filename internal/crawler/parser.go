package crawler

import (
	"bytes"
	"fmt"
	"golang.org/x/net/html"
	"net/url"
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

	return &ParseResult{
		Links: extractLinks(page.URL, node),
	}, nil
}

func extractLinks(currentURL string, node *html.Node) []string {
	var links []string

	if node.Type == html.ElementNode && node.Data == "a" {
		for _, attr := range node.Attr {
			if attr.Key == "href" && attr.Val != "" {
				href := attr.Val

				switch {
				case href == "/":
				case href == "#":
				case href[0] == '#':
					// skip
					continue
				default:
					links = append(links, resolveAbsoluteUrl(currentURL, href))
				}
			}
		}
	}

	// recursive walk
	for c := node.FirstChild; c != nil; c = c.NextSibling {
		links = append(links, extractLinks(currentURL, c)...)
	}

	return links
}

func resolveAbsoluteUrl(currentUrl string, path string) string {
	base, err := url.Parse(currentUrl)
	if err != nil {
		return currentUrl + path
	}

	ref, err := url.Parse(path)
	if err != nil {
		return currentUrl + path
	}

	return base.ResolveReference(ref).String()
}
