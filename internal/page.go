package internal

import (
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
)

type Page struct {
	URL         string
	Content     []byte
	Links       []string
	Stylesheets []string
	Scripts     []string
	Images      []string
}

func Parse(url string, content []byte) (*Page, error) {
	parsed, err := htmlparser.Parse(url, content)
	if err != nil {
		return nil, err
	}

	return &Page{
		URL:         url,
		Content:     content,
		Links:       parsed.Links,
		Stylesheets: parsed.Stylesheets,
		Scripts:     parsed.Scripts,
		Images:      parsed.Images,
	}, nil
}
