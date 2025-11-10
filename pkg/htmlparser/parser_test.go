package htmlparser

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	t.Run("example1", func(t *testing.T) {
		testFile := filepath.Join("../../testdata", "example1.html")

		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("failed to read test testFile %q: %v", testFile, err)
		}

		rootNode, resources, err := ParseHTMLResources(content)
		if err != nil {
			t.Fatalf("failed to parse test testFile %q: %v", testFile, err)
		}

		if rootNode == nil {
			t.Fatalf("failed to parse test testFile %q: root node is nil", testFile)
		}

		var a, css, scripts, imgs []string
		for _, res := range resources {
			tag := res.Node.Data

			switch tag {
			case "a":
				a = append(a, res.SourceURL)
			case "link":
				css = append(css, res.SourceURL)
			case "script":
				scripts = append(scripts, res.SourceURL)
			case "img":
				imgs = append(imgs, res.SourceURL)
			}
		}

		assertSomeResourcesFounds(t, a, []string{
			"https://www.sheldonbrown.com/index.html",
			"web_glossary.html#browser",
			"web_sample1.html#href2",
		})

		assertSomeResourcesFounds(t, css, []string{
			"https://www.sheldonbrown.com/common-data/document.css",
			"https://www.sheldonbrown.com/common-data/screen.css",
			"https://www.sheldonbrown.com/common-data/print.css",
		})

		assertSomeResourcesFounds(t, scripts, []string{
			"https://www.googletagmanager.com/gtag/js?id=G-YRNYST4RX7",
			"http://pagead2.googlesyndication.com/pagead/show_ads.js",
		})

		assertSomeResourcesFounds(t, imgs, []string{
			"https://www.sheldonbrown.com/images/scb_eagle_contact.jpeg",
		})
	})
}

func assertSomeResourcesFounds(t *testing.T, got []string, want []string) {
	for _, w := range want {
		if !slices.Contains(got, w) {
			t.Errorf("no %q found in %v", w, got)
		}
	}
}
