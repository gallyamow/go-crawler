package main

import (
	"context"
	"fmt"
	"github.com/gallyamow/go-crawler/internal"
	"github.com/gallyamow/go-crawler/pkg/httpclient"
	"github.com/gallyamow/go-crawler/pkg/retry"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

func main() {
	config, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration %v", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.SlogValue(),
	}))

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

	startPage, err := internal.NewPage(config.StartURL)
	if err != nil {
		logger.Error("Failed to parse startURL", "err", err, "value", config.StartURL)
		os.Exit(1)
	}

	var httpPool = &sync.Pool{
		New: func() any {
			return httpclient.NewClient(httpclient.WithTimeout(config.Timeout))
		},
	}

	queue := internal.NewQueue(config.MaxConcurrent)
	queue.Push(startPage)

	startedAt := time.Now()
	cnt := 0
	resCh := saveStage(ctx, downloadStage(ctx, queue.Out(), config, httpPool, logger), config, logger)

	for res := range resCh {
		cnt++
		logger.Info("Page saved", "url", res.ResolveRelativeSavePath())
	}

	logger.Info("Crawling completed",
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", cnt,
	)
}

func downloadStage(ctx context.Context, inCh <-chan internal.Downloadable, config *internal.Config, httpClientPool *sync.Pool, logger *slog.Logger) chan internal.Parsable {
	outCh := make(chan internal.Parsable)

	var wg sync.WaitGroup
	wg.Add(config.MaxConcurrent)

	for range config.MaxConcurrent {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inCh:
					if !ok {
						return
					}

					size, err := retry.Retry[int](ctx, func() (int, error) {
						downloadErr := downloadItem(ctx, item, httpClientPool)
						if downloadErr != nil {
							return 0, downloadErr
						}
						return item.GetSize(), nil
					}, retry.NewConfig(retry.WithMaxAttempts(config.RetryAttempts), retry.WithDelay(config.RetryDelay)))

					if err != nil {
						logger.Info(fmt.Sprintf("Item '%s' downloading skipped after %d attempts with error %v.", item.GetURL(), config.RetryAttempts, err))
						continue
					}

					logger.Info(fmt.Sprintf("Item '%s' downloaded, %d bytes.", item.GetURL(), size))

					select {
					case <-ctx.Done():
						return
					case outCh <- item.(internal.Parsable):
					}
				}
			}
		}()
	}

	defer func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh
}

func parseStage(ctx context.Context, inCh <-chan internal.Parsable, config *internal.Config, logger *slog.Logger) chan internal.Downloadable {
	outCh := make(chan internal.Downloadable)

	var wg sync.WaitGroup
	wg.Add(config.MaxConcurrent)

	// concurrently parsing
	for range config.MaxConcurrent {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inCh:
					if !ok {
						return
					}

					children, err := item.ParseChild()
					if err != nil {
						logger.Info(fmt.Sprintf("Item parsing skipped with error %v.", err))
						continue
					}

					logger.Info(fmt.Sprintf("Item parsed."))

					for _, child := range children {
						select {
						case <-ctx.Done():
							return
						case outCh <- child:
						}
					}
				}
			}
		}()
	}

	defer func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh
}

func saveStage(ctx context.Context, inCh <-chan internal.Savable, config *internal.Config, logger *slog.Logger) chan internal.Savable {
	outCh := make(chan internal.Savable)

	var wg sync.WaitGroup
	wg.Add(config.MaxConcurrent)

	for range config.MaxConcurrent {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inCh:
					if !ok {
						return
					}

					path, err := retry.Retry[string](ctx, func() (string, error) {
						path, saveErr := saveItem(ctx, config.OutputDir, item)
						if saveErr != nil {
							return "", saveErr
						}
						return path, nil
					}, retry.NewConfig(retry.WithMaxAttempts(config.RetryAttempts), retry.WithDelay(config.RetryDelay)))

					if err != nil {
						logger.Info(fmt.Sprintf("Item '%s' saving skipped after %d attempts with error %v.", path, config.RetryAttempts, err))
						continue
					}

					logger.Info(fmt.Sprintf("Item '%s' saved.", path))

					select {
					case <-ctx.Done():
						return
					case outCh <- item:
					}
				}
			}
		}()
	}

	defer func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh
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

	item.BeforeSave()

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return savePath, nil
}
