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

		// все URL без anchor
		a := []string{
			"https://www.sheldonbrown.com/index.html",
			"https://www.sheldonbrown.com/web_glossary.html",
			"https://www.sheldonbrown.com/web_sample1.html",
		}
		assertAllUrlFound(t, "a", page.URLMap, a)

		externalA := []string{
			"https://www.external.com/1.html",
			"https://www.google.com/",
			"https://www.ya.ru/some_path",
		}
		assertAllUrlNotFound(t, "a", page.URLMap, externalA)

		css := []string{
			"https://www.sheldonbrown.com/common-data/document.css",
			"https://www.sheldonbrown.com/common-data/screen.css",
			"https://www.sheldonbrown.com/common-data/print.css",
		}
		assertAllUrlFound(t, "link", page.URLMap, css)

		externalCss := []string{
			"https://www.external.com/1.css",
		}
		assertAllUrlFound(t, "link", page.URLMap, externalCss)

		scripts := []string{
			"https://www.sheldonbrown.com/common-data/added.js?someAttr=true",
			"https://www.sheldonbrown.com/common-data/added2.js",
		}
		assertAllUrlFound(t, "script", page.URLMap, scripts)

		externalScripts := []string{
			"https://www.googletagmanager.com/gtag/js?id=G-YRNYST4RX7",
			"http://pagead2.googlesyndication.com/pagead/show_ads.js",
			"https://www.external.com/1.js",
		}
		assertAllUrlFound(t, "script", page.URLMap, externalScripts)

		imgs := []string{
			"https://www.sheldonbrown.com/images/scb_eagle_contact.jpeg",
		}
		assertAllUrlFound(t, "img", page.URLMap, imgs)

		externalImgs := []string{
			"https://www.external.com/1.jpg",
		}
		assertAllUrlFound(t, "img", page.URLMap, externalImgs)
	})
}

func assertAllUrlFound(t *testing.T, tag string, got map[string][]*htmlparser.ResourceNode, want []string) {
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
			t.Errorf("tag %q not found in %v", tag, got[w])
		}
	}
}

func assertAllUrlNotFound(t *testing.T, tag string, got map[string][]*htmlparser.ResourceNode, want []string) {
	for _, w := range want {
		_, ok := got[w]
		if ok {
			t.Errorf("url %q found in %v", w, got)
		}

		found := false
		for _, r := range got[w] {
			if r.Tag == tag {
				found = true
			}
		}

		if found {
			t.Errorf("tag %q found in %v", tag, got[w])
		}
	}
}
