package internal

import (
	"github.com/gallyamow/go-crawler/pkg/htmlparser"
	"os"
	"path/filepath"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("example1", func(t *testing.T) {
		testFile := filepath.Join("../testdata", "example1.html")

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read test testFile %q: %v", testFile, err)
		}

		testUrl := "https://www.sheldonbrown.com/web_sample1.html"
		page, err := Parse("https://www.sheldonbrown.com/web_sample1.html", content)
		if err != nil {
			t.Fatalf("failed to parse test page %q: %v", testUrl, err)
		}

		a := []string{
			"https://www.sheldonbrown.com/index.html",
			"https://www.sheldonbrown.com/web_glossary.html",
			"https://www.sheldonbrown.com/web_sample1.html",
		}
		assertSomeUrlFound(t, "a", page.URLMap, a)

		css := []string{
			"https://www.sheldonbrown.com/common-data/document.css",
			"https://www.sheldonbrown.com/common-data/screen.css",
			"https://www.sheldonbrown.com/common-data/print.css",
		}
		assertSomeUrlFound(t, "link", page.URLMap, css)

		scripts := []string{
			"https://www.googletagmanager.com/gtag/js?id=G-YRNYST4RX7",
			"http://pagead2.googlesyndication.com/pagead/show_ads.js",
		}
		assertSomeUrlFound(t, "script", page.URLMap, scripts)

		imgs := []string{
			"https://www.sheldonbrown.com/images/scb_eagle_contact.jpeg",
		}
		assertSomeUrlFound(t, "img", page.URLMap, imgs)
	})
}

func assertSomeUrlFound(t *testing.T, tag string, got map[string][]*htmlparser.ResourceNode, want []string) {
	for _, w := range want {
		_, ok := got[w]
		if !ok {
			t.Errorf("url %q not found in %v", w, got)
		}

		found := false
		for _, r := range got[w] {
			if r.Tag == tag {
				found = true
			}
		}

		if !found {
			t.Errorf("tag %q found in %v", tag, got[w])
		}
	}
}
