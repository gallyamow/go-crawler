package internal

import (
	"fmt"
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	"golang.org/x/net/html"
	urllib "net/url"
	"path"
)

type Page struct {
	URL      *urllib.URL
	RootNode *html.Node
	Content  []byte
	URLMap   map[string][]*htmlparser.ResourceNode
}

// Parse парсит контент страницы, нормализует ссылки и сохраняет их вместе с оригинальными значениями.
func Parse(rawURL string, content []byte) (*Page, error) {
	pageURL, err := urllib.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %v", err)
	}

	rootNode, resources, err := htmlparser.ParseResources(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page content: %v", err)
	}

	page := Page{
		RootNode: rootNode,
		URL:      pageURL,
		Content:  content,
		URLMap:   buildUrlMap(resources, pageURL),
	}

	return &page, nil
}

func buildUrlMap(resources []*htmlparser.ResourceNode, pageURL *urllib.URL) map[string][]*htmlparser.ResourceNode {
	res := map[string][]*htmlparser.ResourceNode{}

	for _, r := range resources {
		srcURL, err := urllib.Parse(r.Src)
		if err != nil {
			continue
		}

		// skip resources from external domains
		// нужно только для ссылок
		if r.Tag == "a" && srcURL.Host != "" && srcURL.Host != pageURL.Host {
			continue
		}

		// drop anchor
		srcURL.Fragment = ""

		// make absolute
		srcURL = pageURL.ResolveReference(srcURL)
		key := srcURL.String()

		if _, ok := res[key]; !ok {
			res[key] = []*htmlparser.ResourceNode{}
		}

		res[key] = append(res[key], r)
	}

	return res
}

func resolvePath(url *urllib.URL) (string, string) {
	dir := path.Dir(url.Path)
	name := path.Base(url.Path)
	// обработать
	//if name == "." || name == "/" {
	return dir, name
}
