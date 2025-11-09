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

type Savable interface {
	ResolveFilePath() string
	GetContent() []byte
}

type Downloadable interface {
	GetURL() string
	SetContent(content []byte) error
}

type Page struct {
	URL      *urllib.URL
	Content  []byte
	RootNode *html.Node
	Links    []*PageResource
	Assets   []*PageResource
}

func (p *Page) ResolveFilePath() string {
	dir := path.Dir(p.URL.Path)

	name := path.Base(p.URL.Path)
	if name == "." || name == "/" {
		// fallback name
		name = "index"
	}

	return filepath.Join(dir, name) + ".html"
}

func (p *Page) GetContent() []byte {
	return p.Content
}

func (p *Page) GetURL() string {
	return p.URL.String()
}

func (p *Page) SetContent(content []byte) error {
	rootNode, parsedResources, err := htmlparser.ParseResources(content)
	if err != nil {
		return fmt.Errorf("failed to parse page content: %v", err)
	}

	links, assets := resolveAssets(p.URL, parsedResources)

	p.Content = content
	p.RootNode = rootNode
	p.Links = links
	p.Assets = assets

	return nil
}

type PageResource struct {
	Resource *html.Node
	URL      *urllib.URL
	External bool
}

func (r *PageResource) ResolveFilePath() string {
	dir := path.Dir(r.URL.Path)

	var name string
	if r.External {
		name = "ext-" + hasher(r.URL.String())
	} else {
		name = path.Base(r.URL.Path)

		// fallback name
		if name == "." || name == "/" {
			// расширение?
			name = hasher(r.URL.String())
		}
	}

	return filepath.Join(dir, name)
}

func (r *PageResource) GetContent() []byte {
	return []byte{}
}

func hasher(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
