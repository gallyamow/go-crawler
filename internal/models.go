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

type CrawledItem interface {
	GetURL() string
	ResolveSavePath() string
	GetContent() []byte
	SetContent(content []byte) error
	//Child() []CrawledItem
}

type Page struct {
	URL      *urllib.URL
	RootNode *html.Node
	Content  []byte
	Links    []*Link
	Assets   []*Resource
}

func NewPage(rawURL string) (*Page, error) {
	url, err := urllib.Parse(rawURL)

	if err != nil {
		return nil, fmt.Errorf("failed to parse url %q: %v", rawURL, err)
	}
	return &Page{
		URL: url,
	}, nil
}

func (p *Page) GetURL() string {
	return p.URL.String()
}

func (p *Page) ResolveSavePath() string {
	return pagePath(p.URL)
}

func (p *Page) GetContent() []byte {
	return p.Content
}

func (p *Page) SetContent(content []byte) error {
	rootNode, parsedResources, err := htmlparser.ParseHTMLResources(content)

	if err != nil {
		return fmt.Errorf("failed to parse page content: %v", err)
	}

	links, assets := resolveLinksAndAssets(p.URL, parsedResources)

	p.Content = content
	p.RootNode = rootNode
	p.Links = links
	p.Assets = assets

	return nil
}

//func (p *Page) Child() []CrawledItem {
//	var res []CrawledItem
//
//	// @idiomatic: interface slice conversion
//	// так нельзя, срезы разных типов — несовместимы, нужно преобразовать вручную
//	// res = append(res, p.Links...)
//	for _, link := range p.Links {
//		res = append(res, link)
//	}
//
//	for _, asset := range p.Assets {
//		res = append(res, asset)
//	}
//
//	return res
//}

type Link struct {
	URL          *urllib.URL
	HTMLResource *htmlparser.HTMLResource
}

func (l *Link) ResolveSavePath() string {
	return pagePath(l.URL)
}

type Resource struct {
	URL          *urllib.URL
	HTMLResource *htmlparser.HTMLResource
	Content      []byte
}

func (r *Resource) GetURL() string {
	return r.URL.String()
}

func (r *Resource) ResolveSavePath() string {
	dir := path.Dir(r.URL.Path)

	var name string
	name = path.Base(r.URL.Path)

	// fallback name
	if name == "." || name == "/" {
		// расширение?
		name = hasher(r.URL.String())
	}

	return filepath.Join(dir, name)
}

func (r *Resource) GetContent() []byte {
	return r.Content
}

func (r *Resource) SetContent(content []byte) error {
	r.Content = content
	return nil
}

//func (r *Resource) Child() []CrawledItem {
//	return []CrawledItem{}
//}

func hasher(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func pagePath(u *urllib.URL) string {
	dir := path.Dir(u.Path)

	name := path.Base(u.Path)
	if name == "." || name == "/" {
		// fallback name
		name = "index"
	}

	return filepath.Join(dir, name) + ".html"
}
