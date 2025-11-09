package internal

import (
	"fmt"
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	urllib "net/url"
)

// Parse парсит контент страницы, нормализует ссылки и сохраняет их вместе с оригинальными значениями.
func Parse(pageURL *urllib.URL, content []byte) (*Page, error) {
	rootNode, parsedResources, err := htmlparser.ParseHTMLResources(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page content: %v", err)
	}

	links, assets := resolveAssets(pageURL, parsedResources)

	page := &Page{
		URL:     pageURL,
		Content: content,
		Node:    rootNode,
		Links:   links,
		Assets:  assets,
	}

	return page, nil
}

func resolveAssets(pageURL *urllib.URL, parsedResources []*htmlparser.HTMLResource) ([]*Resource, []*Resource) {
	var links []*Resource
	var assets []*Resource

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

			links = append(links, &Resource{
				Node:     p.Node,
				URL:      srcURL,
				External: external,
			})
		} else {
			assets = append(assets, &Resource{
				Node:     p.Node,
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

func buildAssetsURLMapping(prs []*Resource) map[string][]*Resource {
	res := map[string][]*Resource{}

	for _, p := range prs {
		key := p.URL.String()

		if _, ok := res[key]; !ok {
			res[key] = []*Resource{}
		}

		res[key] = append(res[key], p)
	}

	return res
}
