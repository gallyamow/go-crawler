package internal

import (
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	urllib "net/url"
)

// Parse парсит контент страницы, нормализует ссылки и сохраняет их вместе с оригинальными значениями.
//func Parse(pageURL *urllib.sourceURL, content []byte) (*Page, error) {
//	rootNode, parsedResources, err := htmlparser.ParseHTMLResources(content)
//	if err != nil {
//		return nil, fmt.Errorf("failed to parse page content: %v", err)
//	}
//
//	links, assets := resolveLinksAndAssets(pageURL, parsedResources)
//
//	page := &Page{
//		sourceURL:     pageURL,
//		Content: content,
//		HTMLNode:    rootNode,
//		Links:   links,
//		asset:  assets,
//	}
//
//	return page, nil
//}

func resolveLinksAndAssets(pageURL *urllib.URL, htmlResources []*htmlparser.HTMLResource) ([]*Link, []*asset) {
	var links []*Link
	var assets []*asset

	for _, hr := range htmlResources {
		srcURL, err := urllib.Parse(hr.SourceURL)
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
				HTMLNode: hr.Node,
				URL:      srcURL,
			})
		} else {
			assets = append(assets, &asset{
				HTMLNode:  hr.Node,
				sourceURL: srcURL,
			})
		}
	}

	return links, assets
}

//// Transform
//func (p *Page) Transform() {
//	assetsMap := buildAssetsURLMapping(p.asset)
//
//	for key, prs := range assetsMap {
//		for _, p := range prs {
//			p.sourceURL
//		}
//	}
//}

func buildAssetsURLMapping(prs []*asset) map[string][]*asset {
	res := map[string][]*asset{}

	for _, p := range prs {
		key := p.sourceURL.String()

		if _, ok := res[key]; !ok {
			res[key] = []*asset{}
		}

		res[key] = append(res[key], p)
	}

	return res
}
