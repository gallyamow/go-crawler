package main

import (
	"context"
	"fmt"
	"github.com/gallyamow/go-crawler/internal"
	"github.com/gallyamow/go-crawler/pkg/httpclient"
	"log/slog"
	"os"
	"path/filepath"
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

	jobCh := make(chan internal.CrawledItem, maxConcurrent)
	resCh := make(chan internal.CrawledItem, maxConcurrent)
	defer close(jobCh)
	defer close(resCh)

	for i := range maxConcurrent {
		go downloadingWorker(ctx, i, jobCh, resCh, &httpClientPool, logger)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	var cnt = 0
	go func() {
		defer wg.Done()

		cnt = resultsHandler(ctx, jobCh, resCh, baseDir, pagesLimit, logger)
	}()

	// starting
	page, err := internal.NewPage(startUrl)
	if err != nil {
		logger.Error("Failed to parse starting URL", "err", err, "startUrl", startUrl)
	}
	jobCh <- page

	wg.Wait()

	logger.Info("Crawling completed",
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", cnt,
	)
}

func resultsHandler(ctx context.Context, jobCh chan<- internal.CrawledItem, resCh <-chan internal.CrawledItem, baseDir string, pagesLimit int, logger *slog.Logger) int {
	visited := make(map[string]struct{})
	cnt := 0

	for {
		item, ok := <-resCh
		if !ok {
			break
		}

		rawURL := item.GetURL()

		cnt += 1
		if cnt >= pagesLimit {
			logger.Info("Pages count limit exceed", "limit", pagesLimit)
			return cnt
		}

		// queue assets and links downloading
		for _, c := range item.Child() {
			u := c.GetURL()
			if _, seen := visited[u]; seen {
				continue
			}

			select {
			case <-ctx.Done():
				return cnt
			case jobCh <- c:
				visited[u] = struct{}{}
			}
		}

		// concurrently save?
		// it is time to save? page is fully downloaded? anyway we need to transform some page nodes
		err := saveItem(ctx, baseDir, item)
		if err != nil {
			logger.Error("Failed to save", "err", err, "item", item)
			continue
		}

		logger.Info(fmt.Sprintf("Saved %s", rawURL))
	}

	return cnt
}

func downloadingWorker(ctx context.Context, i int, jobCh <-chan internal.CrawledItem, resCh chan<- internal.CrawledItem, httpClientPool *sync.Pool, logger *slog.Logger) {
	for {
		item, ok := <-jobCh
		if !ok {
			return
		}

		if ctx.Err() != nil {
			return
		}

		rawURL := item.GetURL()

		// TODO: retry
		err := downloadItem(ctx, item, httpClientPool)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to download %s, skip?", rawURL), "err", err)
			continue
		}

		logger.Info(fmt.Sprintf("Downloaded %s", rawURL))

		select {
		case <-ctx.Done():
			return
		case resCh <- item:
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

	// TODO: transform content

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}
