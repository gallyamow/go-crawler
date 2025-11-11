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
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	config, err := internal.LoadConfig()
	if err != nil {
		logger.Error("Failed to load configuration", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		logger.Info("Received shutdown signal, stopping crawler...")
		cancel()
	}()

	startedAt := time.Now()

	var httpClientPool = sync.Pool{
		New: func() any {
			return httpclient.NewClient()
		},
	}

	jobCh := make(chan *internal.Page, config.MaxConcurrent)
	resCh := make(chan *internal.Page, config.MaxConcurrent)
	done := make(chan interface{}, 1)

	queue := internal.NewQueue()

	for i := range config.MaxConcurrent {
		go pageWorker(ctx, i, jobCh, resCh, queue, config, &httpClientPool, logger)
	}

	var cnt = 0
	go handleResults(ctx, jobCh, resCh, done, queue, &cnt, config, logger)

	// starting
	page, err := internal.NewPage(config.StartURL)
	if err != nil {
		logger.Error("Failed to parse starting sourceURL", "err", err, "startURL", config.StartURL)
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
func handleResults(ctx context.Context, jobCh chan<- *internal.Page, resCh <-chan *internal.Page, done chan interface{}, queue *internal.DownloadableQueue, cnt *int, config *internal.Config, logger *slog.Logger) {
	defer close(done)
	for {
		page, ok := <-resCh
		if !ok {
			break
		}

		pageURL := page.GetURL()

		*cnt += 1
		if *cnt >= config.MaxCount {
			logger.Info("Pages count limit exceed", "limit", config.MaxCount)
			return
		}

		queue.Ack(page)

		for _, link := range page.Links {
			if ctx.Err() != nil {
				return
			}

			newPage := &internal.Page{
				URL: link.URL,
			}

			if queue.Push(newPage) {
				select {
				case <-ctx.Done():
					return
				case jobCh <- newPage:
				}
			}
		}

		// concurrently save?
		path, err := saveItem(ctx, config.OutputDir, page)
		if err != nil {
			logger.Error("Failed to save", "err", err, "page", page)
			continue
		}

		logger.Info(fmt.Sprintf("Saved %s as %s", pageURL, path))
	}
}

// pageWorker загружает всю страницу целиком, все ее assets и сохраняет все.
func pageWorker(ctx context.Context, i int, jobCh <-chan *internal.Page, resCh chan<- *internal.Page, queue *internal.DownloadableQueue, config *internal.Config, httpClientPool *sync.Pool, logger *slog.Logger) {
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

			assetURL := asset.GetURL()

			err = downloadItem(ctx, asset, httpClientPool)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to download %s", assetURL), "err", err)
				continue
			}

			path, err := saveItem(ctx, config.OutputDir, asset)
			if err != nil {
				logger.Error(fmt.Sprintf("Failed to save %s", assetURL), "err", err)
				continue
			}

			logger.Info(fmt.Sprintf("asset %s saved as %s", assetURL, path))
		}

		// rewrite scr nodes
		pagePath := page.ResolveRelativeSavePath()
		for _, asset := range page.Assets {
			asset.RefreshHTMLNodeURL(pagePath) // makeRelativeURL(pagePath, asset.ResolveRelativeSavePath()))
		}
		for _, link := range page.Links {
			link.RefreshHTMLNodeURL(pagePath) //makeRelativeURL(pagePath, link.ResolveRelativeSavePath()))
		}

		// replace content
		var buf bytes.Buffer
		err = html.Render(&buf, page.HTMLNode)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to transform %s", pageURL), "err", err)
			continue
		}
		page.Content = buf.Bytes()

		path, err := saveItem(ctx, config.OutputDir, page)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to save %s", pageURL), "err", err)
			continue
		}

		logger.Info(fmt.Sprintf("Page saved %s as %s", pageURL, path))

		select {
		case <-ctx.Done():
			return
		case resCh <- page:
		}
	}
}

func downloadItem(ctx context.Context, item internal.Downloadable, httpClientPool *sync.Pool) error {
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

func saveItem(ctx context.Context, baseDir string, item internal.Savable) (string, error) {
	savePath := filepath.Join(baseDir, item.ResolveRelativeSavePath())

	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return "", fmt.Errorf("create directory: %w", err)
	}

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return savePath, nil
}
