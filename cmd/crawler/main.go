package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/gallyamow/go-crawler/internal"
	"github.com/gallyamow/go-crawler/pkg/httpclient"
	"golang.org/x/net/html"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	pagesLimit := 100
	maxConcurrent := 10

	startUrl := "https://go.dev/learn/"
	baseDir := "./.tmp"

	startedAt := time.Now()

	var httpClientPool = sync.Pool{
		New: func() any {
			return httpclient.NewClient()
		},
	}

	jobCh := make(chan *internal.Page, maxConcurrent)
	resCh := make(chan *internal.Page, maxConcurrent)
	done := make(chan interface{}, 1)
	var visited sync.Map

	for i := range maxConcurrent {
		go pageWorker(ctx, i, jobCh, resCh, &visited, baseDir, &httpClientPool, logger)
	}

	var cnt = 0
	go handleResults(ctx, jobCh, resCh, done, &visited, &cnt, baseDir, pagesLimit, logger)

	// starting
	page, err := internal.NewPage(startUrl)
	if err != nil {
		logger.Error("Failed to parse starting URL", "err", err, "startUrl", startUrl)
	}
	jobCh <- page

	// wait
	<-done

	close(jobCh)
	close(resCh)

	logger.Info("Crawling completed",
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", cnt,
	)
}

// handleResults публикует остальные страницы
func handleResults(ctx context.Context, jobCh chan<- *internal.Page, resCh <-chan *internal.Page, done chan interface{}, visited *sync.Map, cnt *int, baseDir string, pagesLimit int, logger *slog.Logger) {
	defer close(done)
	for {
		page, ok := <-resCh
		if !ok {
			break
		}

		pageURL := page.GetURL()

		*cnt += 1
		if *cnt >= pagesLimit {
			logger.Info("Pages count limit exceed", "limit", pagesLimit)
			return
		}

		for _, link := range page.Links {
			linkURL := link.URL.String()
			if _, seen := visited.Load(linkURL); seen {
				continue
			}

			newPage := &internal.Page{
				URL: link.URL,
			}

			select {
			case <-ctx.Done():
				return
			case jobCh <- newPage:
				visited.Store(linkURL, struct{}{})
			}
		}

		// concurrently save?
		err := saveItem(ctx, baseDir, page)
		if err != nil {
			logger.Error("Failed to save", "err", err, "page", page)
			continue
		}

		logger.Info(fmt.Sprintf("Saved %s", pageURL))
	}
}

// pageWorker загружает всю страницу целиком, все ее assets и сохраняет все.
func pageWorker(ctx context.Context, i int, jobCh <-chan *internal.Page, resCh chan<- *internal.Page, visited *sync.Map, baseDir string, httpClientPool *sync.Pool, logger *slog.Logger) {
	for {
		page, ok := <-jobCh
		if !ok {
			return
		}

		if ctx.Err() != nil {
			return
		}

		pageURL := page.GetURL()

		err := downloadItem(ctx, page, httpClientPool)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to download %s", pageURL), "err", err)
			continue
		}

		logger.Info(fmt.Sprintf("Downloaded %s, %d bytes", pageURL, len(page.Content)))

		// download assets
		for _, asset := range page.Assets {
			if ctx.Err() != nil {
				break
			}

			assetURL := asset.URL.String()
			if _, seen := visited.Load(assetURL); seen {
				continue
			}

			err = downloadItem(ctx, asset, httpClientPool)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to download %s", assetURL), "err", err)
				continue
			}

			err = saveItem(ctx, baseDir, asset)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to save %s", assetURL), "err", err)
				continue
			}

			visited.Store(assetURL, struct{}{})
			logger.Info(fmt.Sprintf("Asset saved %s", assetURL))
		}

		// rewrite scr nodes
		pagePath := page.ResolveSavePath()
		for _, asset := range page.Assets {
			asset.HTMLResource.SetSrc(makeRelativeURL(pagePath, asset.ResolveSavePath()))
		}
		for _, link := range page.Links {
			link.HTMLResource.SetSrc(makeRelativeURL(pagePath, link.ResolveSavePath()))
		}

		// replace content
		var buf bytes.Buffer
		err = html.Render(&buf, page.RootNode)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to transform %s", pageURL), "err", err)
			continue
		}
		page.Content = buf.Bytes()

		err = saveItem(ctx, baseDir, page)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to save %s", pageURL), "err", err)
			continue
		}

		logger.Info(fmt.Sprintf("Page saved %s", pageURL))

		select {
		case <-ctx.Done():
			return
		case resCh <- page:
		}
	}
}

func downloadItem(ctx context.Context, item internal.CrawledItem, httpClientPool *sync.Pool) error {
	httpClient := httpClientPool.Get().(*httpclient.Client)
	defer httpClientPool.Put(httpClient)

	// buffered?
	// check max size limit and extension
	content, err := httpClient.Get(ctx, item.GetURL())
	if err != nil {
		return err
	}

	err = item.SetContent(content)
	if err != nil {
		return err
	}

	return nil
}

func saveItem(ctx context.Context, baseDir string, item internal.CrawledItem) error {
	savePath := filepath.Join(baseDir, item.ResolveSavePath())

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

func makeRelativeURL(pagePath, assetPath string) string {
	fromDir := filepath.Dir(pagePath)
	rel, err := filepath.Rel(fromDir, assetPath)

	// fallback
	if err != nil {
		return "./" + filepath.Base(assetPath)
	}

	// replace slashes
	rel = strings.ReplaceAll(rel, string(filepath.Separator), "/")
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}

	return rel
}
