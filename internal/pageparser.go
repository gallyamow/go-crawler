package internal

import (
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	urllib "net/url"
)

// Parse парсит контент страницы, нормализует ссылки и сохраняет их вместе с оригинальными значениями.
//func Parse(pageURL *urllib.URL, content []byte) (*Page, error) {
//	rootNode, parsedResources, err := htmlparser.ParseHTMLResources(content)
//	if err != nil {
//		return nil, fmt.Errorf("failed to parse page content: %v", err)
//	}
//
//	links, assets := resolveLinksAndAssets(pageURL, parsedResources)
//
//	page := &Page{
//		URL:     pageURL,
//		Content: content,
//		RootNode:    rootNode,
//		Links:   links,
//		Assets:  assets,
//	}
//
//	return page, nil
//}

func resolveLinksAndAssets(pageURL *urllib.URL, htmlResources []*htmlparser.HTMLResource) ([]*Link, []*Resource) {
	var links []*Link
	var assets []*Resource

	for _, hr := range htmlResources {
		srcURL, err := urllib.Parse(hr.Src)
		if err != nil {
			continue
		}

		// drop anchor
		srcURL.Fragment = ""

		// make absolute
		srcURL = pageURL.ResolveReference(srcURL)

		// проверять можно только после ResolveReference
		if srcURL.Host != pageURL.Host {
			continue
		}

		if hr.Tag() == "a" {
			// skip resources from external domains
			if hr.Tag() == "a" && srcURL.Host != "" && srcURL.Host != pageURL.Host {
				continue
			}

			links = append(links, &Link{
				HTMLResource: hr,
				URL:          srcURL,
			})
		} else {
			assets = append(assets, &Resource{
				HTMLResource: hr,
				URL:          srcURL,
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
