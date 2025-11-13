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
		//Level: config.SlogValue(),
		Level: slog.LevelDebug,
	}))

	// @idiomatic: graceful shutdown (modern way)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// @idiomatic: graceful shutdown (old way)
	/*
		sigChan := make(chan os.Signal, 1)
		// @idiomatic: pass channel to func than writes to it
		// (SIGKILL - не обрабатывают, потому что ядро немедленно завершает процесс)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigChan
			logger.Info("Received shutdown signal, stopping crawler...")
			cancel()
		}()
	*/

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

	queue := internal.NewQueue(config.MaxCount, config.MaxConcurrent, logger)

	// building pipeline
	// should move stages to internal.Queue after renaming internal.QueueManager
	resCh := saveStage(
		ctx,
		parseStage(
			ctx,
			downloadStage(
				ctx,
				queue.Out(), config, httpPool, logger,
			),
			queue, config, logger,
		),
		config,
		logger,
	)

	startedAt := time.Now()
	queue.Push(startPage)

	//time.Sleep(time.Second)

	var pagesCnt, assetsCnt = 0, 0

	for item := range resCh {
		logId := item.(internal.Identifiable).ItemId()
		queue.Ack(item)

		switch item.(type) {
		case *internal.Page:
			pagesCnt++
			logger.Info(fmt.Sprintf("Done for page %d of %d", pagesCnt, config.MaxCount))
		default:
			assetsCnt++
			logger.Info(fmt.Sprintf("Done for asset '%s'", logId))
		}
	}

	logger.Info("Crawling completed",
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", pagesCnt, "assets_crawled", assetsCnt,
	)
}

func downloadStage(ctx context.Context, inCh <-chan internal.Identifiable, config *internal.Config, httpClientPool *sync.Pool, logger *slog.Logger) chan internal.Identifiable {
	// @idiomatic: используем буферизированные каналы, чтобы сгладить отличающуюся скорость каждой стадий
	// (например download -долгий, parse - быстрый)
	outCh := make(chan internal.Identifiable, config.MaxConcurrent)

	var wg sync.WaitGroup
	wg.Add(config.MaxConcurrent)

	for range config.MaxConcurrent {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inCh:
					if !ok {
						return
					}

					logId := item.(internal.Identifiable).ItemId()
					logger.Debug(fmt.Sprintf("Item '%s' received by the 'download' stage", logId))

					downloadableItem := item.(internal.Downloadable)
					size, err := retry.Retry[int](ctx, func() (int, error) {
						downloadErr := downloadItem(ctx, downloadableItem, httpClientPool)
						if downloadErr != nil {
							return 0, downloadErr
						}
						return downloadableItem.GetSize(), nil
					}, retry.NewConfig(retry.WithMaxAttempts(config.RetryAttempts), retry.WithDelay(config.RetryDelay)))

					if err != nil {
						logger.Debug(fmt.Sprintf("Item '%s' downloading skipped, after %d attempts, with error %v.", logId, config.RetryAttempts, err))
						continue
					}

					logger.Debug(fmt.Sprintf("Item '%s' downloaded, size %d bytes.", logId, size))

					select {
					case <-ctx.Done():
						return
					case outCh <- item:
						logger.Debug(fmt.Sprintf("Item '%s' transmitted from the 'download' stage to next one.", logId))
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh
}

func parseStage(ctx context.Context, inCh <-chan internal.Identifiable, queue *internal.Queue, config *internal.Config, logger *slog.Logger) chan internal.Identifiable {
	outCh := make(chan internal.Identifiable, config.MaxConcurrent)

	var wg sync.WaitGroup
	wg.Add(config.MaxConcurrent)

	// concurrently parsing
	for range config.MaxConcurrent {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inCh:

					if !ok {
						return
					}

					logId := item.(internal.Identifiable).ItemId()
					logger.Debug(fmt.Sprintf("Item '%s' received by the 'parse' stage", logId))

					if parsable, ok := item.(internal.Parsable); ok {
						err := parsable.Parse()
						if err != nil {
							logger.Debug(fmt.Sprintf("Item '%s' parsing skipped, with error %v.", logId, err))
							continue
						}

						logger.Debug(fmt.Sprintf("Item '%s' parsed, found child items %d", logId, len(parsable.GetChildren())))

						for _, child := range parsable.GetChildren() {
							queue.Push(child)
						}
					}

					// anyway we should pass item to the next stage
					select {
					case <-ctx.Done():
						return
					case outCh <- item:
						logger.Debug(fmt.Sprintf("Item '%s' transmitted from the 'parse' stage to next one.", logId))
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(outCh)
	}()

	return outCh
}

func saveStage(ctx context.Context, inCh <-chan internal.Identifiable, config *internal.Config, logger *slog.Logger) chan internal.Identifiable {
	outCh := make(chan internal.Identifiable, config.MaxConcurrent)

	var wg sync.WaitGroup
	wg.Add(config.MaxConcurrent)

	for range config.MaxConcurrent {
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-inCh:
					if !ok {
						return
					}

					logId := item.(internal.Identifiable).ItemId()
					logger.Debug(fmt.Sprintf("Item '%s' received by the 'save' stage", logId))

					path, err := retry.Retry[string](ctx, func() (string, error) {
						p, saveErr := saveItem(ctx, config.OutputDir, item.(internal.Savable))
						if saveErr != nil {
							return "", saveErr
						}
						return p, nil
					}, retry.NewConfig(retry.WithMaxAttempts(config.RetryAttempts), retry.WithDelay(config.RetryDelay)))

					if err != nil {
						logger.Debug(fmt.Sprintf("Item '%s' saving skipped, after %d attempts, with error %v.", logId, config.RetryAttempts, err))
						continue
					}

					logger.Debug(fmt.Sprintf("Item '%s' saved to '%s'.", logId, path))

					select {
					case <-ctx.Done():
						return
					case outCh <- item.(internal.Identifiable):
						logger.Debug(fmt.Sprintf("Item '%s' transmitted from the 'save' stage to next one.", logId))
					}
				}
			}
		}()
	}

	go func() {
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

	if transformable, ok := item.(internal.Transformable); ok {
		transformable.Transform()
	}

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return savePath, nil
}
