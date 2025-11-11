package internal

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	"golang.org/x/net/html"
	urllib "net/url"
	pathlib "path"
	"path/filepath"
	"strings"
)

type Savable interface {
	ResolveRelativeSavePath() string
	BeforeSave()
	GetContent() []byte
}

type Downloadable interface {
	GetURL() string
	SetContent(content []byte) error
	GetSize() int
}

type Parsable interface {
	ParseChild() ([]Downloadable, error)
}

type Page struct {
	URL      *urllib.URL
	HTMLNode *html.Node
	Content  []byte
	Links    []*Link
	Assets   []*asset
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

func (p *Page) ResolveRelativeSavePath() string {
	return resolveLocalSavePath(p.URL, "index", "html")
}

func (p *Page) GetContent() []byte {
	return p.Content
}

func (p *Page) GetURL() string {
	return p.URL.String()
}

func (p *Page) GetSize() int {
	return len(p.Content)
}

func (p *Page) SetContent(content []byte) error {
	p.Content = content
	return nil
}

// BeforeSave переписывает src ресурсов relative относительно директории страницы значением.
func (p *Page) BeforeSave() {
	pagePath := p.ResolveRelativeSavePath()

	for _, asset := range p.Assets {
		newURL := makeRelativeURL(pagePath, asset.ResolveRelativeSavePath())
		htmlparser.WriteResourceURL(asset.HTMLNode, newURL)
	}

	for _, link := range p.Links {
		newURL := makeRelativeURL(pagePath, resolveLocalSavePath(link.URL, "", "html"))
		htmlparser.WriteResourceURL(link.HTMLNode, newURL)
	}
}

func (p *Page) ParseChild() ([]Downloadable, error) {
	rootNode, parsedResources, err := htmlparser.ParseHTMLResources(p.GetContent())

	if err != nil {
		return nil, fmt.Errorf("failed to parse page content: %v", err)
	}

	links, assets := resolveLinksAndAssets(p.URL, parsedResources)

	p.HTMLNode = rootNode
	p.Links = links
	p.Assets = assets

	var res []Downloadable

	// @idiomatic: interface slice conversion
	// (append(res, p.Links...) -  нельзя, срезы разных типов — несовместимы, нужно преобразовать вручную)
	for _, l := range p.Links {
		page, err := NewPage(l.URL.String())
		if err != nil {
			// todo log
			continue
		}
		res = append(res, page)
	}

	for _, a := range p.Assets {
		res = append(res, a)
	}

	return res, nil
}

type Link struct {
	URL      *urllib.URL
	HTMLNode *html.Node
}

type CssFile struct {
	asset
}
type ScriptFile struct {
	asset
}
type ImageFile struct {
	asset
}

type asset struct {
	sourceURL *urllib.URL
	HTMLNode  *html.Node
	Content   []byte
}

func (r *asset) GetURL() string {
	return r.sourceURL.String()
}

func (r *asset) GetSize() int {
	return len(r.Content)
}

func (r *asset) ResolveRelativeSavePath() string {
	return resolveLocalSavePath(r.sourceURL, "", "")
}

func (r *asset) GetContent() []byte {
	return r.Content
}

func (r *asset) SetContent(content []byte) error {
	r.Content = content
	return nil
}

func (r *asset) BeforeSave() {
	// noting to do
}

func hasher(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

func resolveLocalSavePath(url *urllib.URL, fallbackName string, ext string) string {
	dir := pathlib.Dir(url.Path)
	name := pathlib.Base(url.Path)

	if name == "." || name == "/" {
		name = fallbackName
	}

	if name == "" {
		name = hasher(url.String())
	}

	path := filepath.Join(dir, name)
	if ext != "" {
		path += "." + ext
	}
	return path

}

func makeRelativeURL(rootPath, localPath string) string {
	fromDir := filepath.Dir(rootPath)
	rel, err := filepath.Rel(fromDir, localPath)

	// fallback
	if err != nil {
		return "./" + filepath.Base(localPath)
	}

	// replace slashes
	rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}

	return rel
}
