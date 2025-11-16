package main

import (
	"context"
	"fmt"
	"github.com/gallyamow/go-crawler/internal"
	"github.com/gallyamow/go-crawler/pkg/fanin"
	"github.com/gallyamow/go-crawler/pkg/httpclient"
	"github.com/gallyamow/go-crawler/pkg/retry"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
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

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: config.SlogValue(),
		//Level: slog.LevelDebug,
	}))

	// @idiomatic: graceful shutdown (modern way)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	startPage, err := internal.NewPage(config.URL)
	if err != nil {
		logger.Error("Failed to parse startURL", "err", err, "value", config.URL)
		os.Exit(1)
	}

	var httpPool = &sync.Pool{
		New: func() any {
			return httpclient.NewClient(httpclient.WithTimeout(config.Timeout))
		},
	}

	// Размеры буферов будем рассчитывать на этой основе
	maxConcurrent := config.MaxConcurrent

	queue := internal.NewQueue(ctx, config.MaxCount, maxConcurrent, logger)

	// @idiomatic: используем буферизированные каналы разных размеров и разное кол-во workers, чтобы регулировать back pressure.
	// На практике bufferSize = workersCnt - часто недостаточно. Обычно используют x2, x4 - ПЕРЕД медленным.
	// Это позволяет стадиям до медленного, выполнять свою работу, а не ждать.
	// В медленном stage, если он IO-bound, то можно увеличить concurrency.
	pagesCh := saveStage(
		ctx,
		parseStage(
			ctx,
			downloadStage(
				ctx,
				queue.Pages(),
				maxConcurrent, maxConcurrent*2,
				config, httpPool, logger,
			),
			maxConcurrent, maxConcurrent*2,
			queue, config, logger,
		),
		maxConcurrent, maxConcurrent*2,
		config,
		logger,
	)

	assetsCh := saveStage(
		ctx,
		downloadStage(
			ctx,
			queue.Assets(),
			maxConcurrent, maxConcurrent*2,
			config, httpPool, logger,
		),
		maxConcurrent, maxConcurrent*2,
		config,
		logger,
	)

	startedAt := time.Now()
	queue.Push(startPage)

	var pagesCnt, assetsCnt = 0, 0

	// @idiomatic: using fan-in to merge channels instead of using for + flags
	for item := range fanin.Merge(pagesCh, assetsCh) {
		switch item.(type) {
		case *internal.Page:
			pagesCnt++
			queue.Ack(item)

			logger.Info(fmt.Sprintf("Done for page %d of %d", pagesCnt, config.MaxCount))
		default:
			assetsCnt++
			logId := item.(internal.Queueable).ItemId()
			queue.Ack(item)

			logger.Info(fmt.Sprintf("Done '%s'", logId))
		}
	}

	msg := "Crawling completed"
	if ctx.Err() != nil {
		msg = "Crawling interrupted"
	}

	logger.Info(
		msg,
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", pagesCnt,
		"assets_crawled", assetsCnt,
	)
}

func downloadStage(ctx context.Context, inCh <-chan internal.Queueable, workersCnt int, bufferSize int, config *internal.Config, httpClientPool *sync.Pool, logger *slog.Logger) chan internal.Queueable {
	outCh := make(chan internal.Queueable, bufferSize)

	var wg sync.WaitGroup
	wg.Add(workersCnt)

	for range workersCnt {
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

					logId := item.(internal.Queueable).ItemId()
					logger.Debug(fmt.Sprintf("Item '%s' received by the 'download' stage", logId))

					downloadableItem := item.(internal.Downloadable)
					size, err := retry.Retry[int](ctx, func() (int, error) {
						downloadErr := downloadItem(ctx, downloadableItem, config, httpClientPool)
						if downloadErr != nil {
							return 0, downloadErr
						}
						return downloadableItem.GetSize(), nil
					}, retry.NewConfig(retry.WithMaxAttempts(config.RetryAttempts), retry.WithDelay(config.RetryDelay)))

					if err != nil {
						logger.Debug(fmt.Sprintf("Item '%s' downloading skipped, after %d attempts, with error: %v.", logId, config.RetryAttempts, err))
						item.SetSkipped("download")
					} else {
						logger.Debug(fmt.Sprintf("Item '%s' downloaded, size %d bytes.", logId, size))
					}

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

func parseStage(ctx context.Context, inCh <-chan internal.Queueable, workersCnt int, bufferSize int, queue *internal.Queue, config *internal.Config, logger *slog.Logger) chan internal.Queueable {
	outCh := make(chan internal.Queueable, bufferSize)

	var wg sync.WaitGroup
	wg.Add(workersCnt)

	// concurrently parsing
	for range workersCnt {
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

					logId := item.(internal.Queueable).ItemId()
					logger.Debug(fmt.Sprintf("Item '%s' received by the 'parse' stage", logId))

					if parsable, ok := item.(internal.Parsable); ok {
						err := parsable.Parse()
						if err != nil {
							logger.Debug(fmt.Sprintf("Item '%s' parsing skipped, with error: %v.", logId, err))
							item.SetSkipped("parse")
						} else {
							logger.Debug(fmt.Sprintf("Item '%s' parsed, found child items %d", logId, len(parsable.GetChildren())))
						}

						// @idiomatic: check context before long-running operations
						if ctx.Err() != nil {
							return
						}

						// Без специальной обработки, добавление в pipeline новых элементов из самого же этого pipeline
						// - всегда будет блокироваться.
						// Суть проблемы:
						// На этой стадии мы добавляем элементы в queue до заполнения его буфера, после этого блокируемся на добавлении очередного child элемента.
						// Но и не отпускаем pipeline на обработку след. элемента.
						//
						// Решение: 1) Вынести в буфер и добавлять его на последней стадии. Но тогда будет просто блокироваться последняя стадия.
						// Решение: 2) Дать это на управление в queue, добавлять в buffer и queue  в отдельной горутине будет
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

func saveStage(ctx context.Context, inCh <-chan internal.Queueable, workersCnt int, bufferSize int, config *internal.Config, logger *slog.Logger) chan internal.Queueable {
	// disk ops too slow, maybe we need more workers?
	outCh := make(chan internal.Queueable, bufferSize)

	var wg sync.WaitGroup
	wg.Add(workersCnt)

	for range workersCnt {
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

					logId := item.(internal.Queueable).ItemId()
					logger.Debug(fmt.Sprintf("Item '%s' received by the 'save' stage", logId))

					path, err := retry.Retry[string](ctx, func() (string, error) {
						p, saveErr := saveItem(ctx, config.OutputDir, item.(internal.Savable))
						if saveErr != nil {
							return "", saveErr
						}
						return p, nil
					}, retry.NewConfig(retry.WithMaxAttempts(config.RetryAttempts), retry.WithDelay(config.RetryDelay)))

					if err != nil {
						logger.Debug(fmt.Sprintf("Item '%s' saving skipped, after %d attempts, with error: %v.", logId, config.RetryAttempts, err))
						item.SetSkipped("save")
					} else {
						logger.Debug(fmt.Sprintf("Item '%s' saved to '%s'.", logId, path))
					}

					select {
					case <-ctx.Done():
						return
					case outCh <- item.(internal.Queueable):
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

func downloadItem(ctx context.Context, item internal.Downloadable, config *internal.Config, httpClientPool *sync.Pool) error {
	client := httpClientPool.Get().(*httpclient.Client)
	defer httpClientPool.Put(client)

	// 1) Try a HEAD request before GET. However, some servers return Content-Length: 0
	//    or the URL provides a stream-like response.
	// 2) If HEAD is not supported or doesn't provide a valid size, read the GET response
	// 	  and stop when the size limit is exceeded.
	head, err := client.Head(ctx, item.GetURL())
	if err != nil {
		return err
	}

	contentLenHeader := head.Header.Get("Content-Length")
	if contentLenHeader != "" {
		size, err := strconv.ParseInt(head.Header.Get("Content-Length"), 10, 64)
		if err == nil && size > config.MaxFileSize {
			return fmt.Errorf("content size exceeds limit: %d", config.MaxFileSize)
		}
	}

	content, err := client.Get(ctx, item.GetURL())
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
		err := transformable.Transform()
		if err != nil {
			return "", fmt.Errorf("transform file: %w", err)
		}
	}

	if err := os.WriteFile(savePath, item.GetContent(), 0644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return savePath, nil
}
