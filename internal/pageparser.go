package internal

import (
	"fmt"
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	urllib "net/url"
)

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
