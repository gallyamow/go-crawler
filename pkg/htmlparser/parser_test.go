package htmlparser

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("Parse", func(t *testing.T) {
		file := filepath.Join("testdata", "example1.html")
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read test file: %v", err)
		}

		res, err := Parse("https://www.sheldonbrown.com/web_sample1.html", content)
		if err != nil {
			t.Errorf("got error: %v", err)
		}

		// we ignore anchors
		assertStringsEqual(t, res.Links, []string{
			"https://www.sheldonbrown.com/index.html",
			"https://www.sheldonbrown.com/web_glossary.html",
			"https://www.sheldonbrown.com/web_sample1.html",
		})

		assertStringsEqual(t, res.Stylesheets, []string{
			"https://www.sheldonbrown.com/common-data/document.css",
			"https://www.sheldonbrown.com/common-data/screen.css",
			"https://www.sheldonbrown.com/common-data/print.css",
		})

		assertStringsEqual(t, res.Scripts, []string{
			"https://www.googletagmanager.com/gtag/js?id=G-YRNYST4RX7",
			"http://pagead2.googlesyndication.com/pagead/show_ads.js",
		})

		assertStringsEqual(t, res.Images, []string{
			"https://www.sheldonbrown.com/images/scb_eagle_contact.jpeg",
		})
	})
}

func assertStringsEqual(t *testing.T, got []string, want []string) {
	slices.Sort(want)
	slices.Sort(got)

	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
