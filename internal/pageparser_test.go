package internal

import (
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
		page, _ := NewPage(testUrl)

		err = page.SetContent(content)
		if err != nil {
			t.Fatalf("failed to parse test page %q: %v", testUrl, err)
		}

		var gotLinks []string

		// все sourceURL без anchor
		internalLinks := []string{
			"https://www.sheldonbrown.com/index.html",
			"https://www.sheldonbrown.com/web_glossary.html",
			"https://www.sheldonbrown.com/web_sample1.html",
		}

		gotLinks = []string{}
		for _, l := range page.Links {
			gotLinks = append(gotLinks, l.URL.String())
		}
		assertAllUrlsFound(t, gotLinks, internalLinks)

		externalLinks := []string{
			"https://www.external.com/1.html",
			"https://www.google.com/",
			"https://www.ya.ru/some_path",
		}
		assertAllUrlsNotFound(t, gotLinks, externalLinks)

		gotAssets := []string{}
		for _, l := range page.Assets {
			gotLinks = append(gotAssets, l.GetURL())
		}

		internalCss := []string{
			"https://www.sheldonbrown.com/common-data/document.css",
			"https://www.sheldonbrown.com/common-data/screen.css",
			"https://www.sheldonbrown.com/common-data/print.css",
		}

		assertAllUrlsFound(t, gotAssets, internalCss)

		externalCss := []string{
			"https://www.external.com/1.css",
		}
		assertAllUrlsNotFound(t, gotAssets, externalCss)

		internalScripts := []string{
			"https://www.sheldonbrown.com/common-data/added.js?someAttr=true",
			"https://www.sheldonbrown.com/common-data/added2.js",
		}

		assertAllUrlsFound(t, gotAssets, internalScripts)

		externalScripts := []string{
			"https://www.googletagmanager.com/gtag/js?id=G-YRNYST4RX7",
			"http://pagead2.googlesyndication.com/pagead/show_ads.js",
			"https://www.external.com/1.js",
		}
		assertAllUrlsNotFound(t, gotAssets, externalScripts)

		imgs := []string{
			"https://www.sheldonbrown.com/images/scb_eagle_contact.jpeg",
		}
		assertAllUrlsFound(t, gotAssets, imgs)

		externalImgs := []string{
			"https://www.external.com/1.jpg",
		}
		assertAllUrlsNotFound(t, gotAssets, externalImgs)
	})
}

func assertAllUrlsFound(t *testing.T, got []string, want []string) {
	for _, w := range want {
		found := false
		for _, g := range got {
			if g == w {
				found = true
			}
		}

		if !found {
			t.Errorf("url %q not found in %v", w, got)
		}
	}
}

func assertAllUrlsNotFound(t *testing.T, got []string, want []string) {
	for _, w := range want {
		found := false
		for _, g := range got {
			if g == w {
				found = true
			}
		}

		if found {
			t.Errorf("url %q found in %v", w, got)
		}
	}
}
