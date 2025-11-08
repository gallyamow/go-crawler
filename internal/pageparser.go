package internal

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	"golang.org/x/net/html"
	urllib "net/url"
	"path"
	"path/filepath"
)

type Page struct {
	URL      *urllib.URL
	Path     string
	Content  []byte
	RootNode *html.Node
	Links    []*PageResource
	Assets   []*PageResource
}

type PageResource struct {
	Resource *html.Node
	URL      *urllib.URL
	External bool
}

// Parse парсит контент страницы, нормализует ссылки и сохраняет их вместе с оригинальными значениями.
func Parse(rawURL string, content []byte) (*Page, error) {
	pageURL, err := urllib.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse url: %v", err)
	}

	rootNode, parsedResources, err := htmlparser.ParseResources(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page content: %v", err)
	}

	links, assets := resolveAssets(pageURL, parsedResources)

	page := &Page{
		URL:      pageURL,
		Content:  content,
		RootNode: rootNode,
		Links:    links,
		Assets:   assets,
	}
	page.Path = resolvePagePath(page)

	return page, nil
}

func resolveAssets(pageURL *urllib.URL, parsedResources []*htmlparser.ResourceNode) ([]*PageResource, []*PageResource) {
	var links []*PageResource
	var assets []*PageResource

	for _, p := range parsedResources {
		srcURL, err := urllib.Parse(p.Src)
		if err != nil {
			continue
		}

		external := srcURL.Host == pageURL.Host

		// drop anchor
		srcURL.Fragment = ""

		// make absolute
		srcURL = pageURL.ResolveReference(srcURL)

		if p.Tag() == "a" {
			// skip resources from external domains
			if p.Tag() == "a" && srcURL.Host != "" && srcURL.Host != pageURL.Host {
				continue
			}

			links = append(links, &PageResource{
				Resource: p.Node,
				URL:      srcURL,
				External: external,
			})
		} else {
			assets = append(assets, &PageResource{
				Resource: p.Node,
				URL:      srcURL,
				External: external,
			})
		}
	}

	return links, assets
}

//// Transform
//func (p *Page) Transform() {
//	assetsMap := buildAssetsURLMapping(p.Assets)
//
//	for key, prs := range assetsMap {
//		for _, p := range prs {
//			p.URL
//		}
//	}
//}

//// Transform
//func (p *Page) Transform() {
//	assetsMap := buildAssetsURLMapping(p.Assets)
//
//	for key, prs := range assetsMap {
//		for _, p := range prs {
//			p.URL
//		}
//	}
//}

func buildAssetsURLMapping(prs []*PageResource) map[string][]*PageResource {
	res := map[string][]*PageResource{}

	for _, p := range prs {
		key := p.URL.String()

		if _, ok := res[key]; !ok {
			res[key] = []*PageResource{}
		}

		res[key] = append(res[key], p)
	}

	return res
}

func resolvePagePath(pr *Page) string {
	dir := path.Dir(pr.URL.Path)

	name := path.Base(pr.URL.Path)
	if name == "." || name == "/" {
		// fallback name
		name = "index"
	}

	return filepath.Join(dir, name) + ".html"
}

func resolveFilePath(pr *PageResource) string {
	dir := path.Dir(pr.URL.Path)

	var name string
	if pr.External {
		name = "ext-" + hasher(pr.URL.String())
	} else {
		name = path.Base(pr.URL.Path)

		// fallback name
		if name == "." || name == "/" {
			// расширение?
			name = hasher(pr.URL.String())
		}
	}

	return filepath.Join(dir, name)
}

func hasher(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
