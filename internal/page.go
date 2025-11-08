package internal

import (
	"fmt"
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	urllib "net/url"
)

type Page struct {
	URL     *urllib.URL
	Content []byte
	URLMap  map[string][]*htmlparser.ResourceNode
}

// Parse парсит контент страницы, нормализует ссылки и сохраняет их вместе с оригинальными значениями.
func Parse(rawURL string, content []byte) (*Page, error) {
	pageURL, err := urllib.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %v", err)
	}

	resources, err := htmlparser.ParseResources(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page content: %v", err)
	}

	urlMap := map[string][]*htmlparser.ResourceNode{}
	for _, r := range resources {
		bareSrc, ok := normalizedSrc(r.Src, pageURL)
		if !ok {
			continue
		}

		if _, ok := urlMap[bareSrc]; !ok {
			urlMap[bareSrc] = []*htmlparser.ResourceNode{}
		}

		urlMap[bareSrc] = append(urlMap[bareSrc], r)
	}

	page := Page{
		URL:     pageURL,
		Content: content,
		URLMap:  urlMap,
	}

	return &page, nil
}

// normalizedSrc удаляем anchor, делаем абсолютным.
func normalizedSrc(src string, pageURL *urllib.URL) (string, bool) {
	srcURL, err := urllib.Parse(src)
	if err != nil {
		return "", false
	}

	srcURL.Fragment = ""

	return pageURL.ResolveReference(srcURL).String(), true
}
