package main

import (
	"context"
	"fmt"
	"github.com/gallyamow/go-crawler/internal"
	"github.com/gallyamow/go-crawler/pkg/httpclient"
	"log/slog"
	urllib "net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	maxCount := 100
	maxConcurrent := 10
	startUrl := "https://go.dev/learn/"
	baseDir := "./.tmp"
	startedAt := time.Now()

	visited := make(map[string]struct{})
	cnt := 0

	var httpClientPool = sync.Pool{
		New: func() any {
			return httpclient.NewClient()
		},
	}

	jobs := make(chan string, maxConcurrent)
	results := make(chan *internal.Page, maxConcurrent)

	for i := range maxConcurrent {
		go worker(ctx, i, baseDir, jobs, results, &httpClientPool, logger)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		for page := range results {
			cnt += 1
			if cnt >= maxCount {
				logger.Info("Page limit exceed", "limit", maxCount)
				return
			}

			for _, link := range page.Links {
				if link.External {
					continue
				}

				url := link.URL.String()

				if _, ok := visited[url]; ok {
					continue
				}

				visited[url] = struct{}{}

				if cnt < maxCount {
					select {
					case jobs <- url:
					default: // Skip if job queue is full
					}
				}
			}
		}
	}()

	jobs <- startUrl

	<-done
	close(jobs)
	close(results)

	logger.Info("Crawling completed",
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", cnt,
	)
}

func worker(ctx context.Context, i int, baseDir string, jobs <-chan string, results chan<- *internal.Page, httpClientPool *sync.Pool, logger *slog.Logger) {
	logger.Info(fmt.Sprintf("Worker %d started", i))

	for url := range jobs {
		logger.Info(fmt.Sprintf("Worker %d is handling %s", i, url))

		pageURL, err := urllib.Parse(url)
		if err != nil {
			logger.Error("Failed to parse url", "err", err, "url", url)
			continue
		}

		page := &internal.Page{URL: pageURL}

		err = downloadItem(ctx, page, httpClientPool)
		if err != nil {
			logger.Error("Failed to download", "err", err, "url", url)
			continue
		}

		// queue assets downloading
		// transform page nodes

		err = saveItem(ctx, baseDir, page)
		if err != nil {
			logger.Error("Failed to save", "err", err, "url", url)
			continue
		}

		logger.Info("handled", "url", url, "saved", "PATH")

		results <- page
	}
}

func downloadItem(ctx context.Context, item internal.Downloadable, httpClientPool *sync.Pool) error {
	httpClient := httpClientPool.Get().(*httpclient.Client)
	defer httpClientPool.Put(httpClient)

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

func saveItem(ctx context.Context, baseDir string, item internal.Savable) error {
	savePath := filepath.Join(baseDir, item.ResolveFilePath())

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// TODO: transform content

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
