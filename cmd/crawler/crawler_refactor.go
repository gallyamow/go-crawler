package main

import (
	"context"
	"fmt"
	"github.com/gallyamow/go-crawler/internal"
	"github.com/gallyamow/go-crawler/pkg/httpclient"
	"github.com/gallyamow/go-crawler/pkg/retry"
	"golang.org/x/sync/errgroup"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// По-красоте: однозначный, backpressure-aware, idiomatic pipeline на errgroup.
// Ключевые идеи:
// - отдельные ветки для страниц и ассетов
// - четкие входные/выходные каналы и ответственность за их закрытие
// - контекст + errgroup для отмены и ожидания завершения
// - контролируемый пул воркеров (MaxConcurrent)

func main() {
	config, err := internal.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load configuration %v", err)
		os.Exit(1)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx, stop := signalWithCancel()
	defer stop()

	g, ctx := errgroup.WithContext(ctx)

	// channels (buffers to smooth differences)
	pageDownloadCh := make(chan internal.Queable, config.MaxConcurrent*2)
	assetDownloadCh := make(chan internal.Queable, config.MaxConcurrent*2)
	downloadedPagesCh := make(chan internal.Queable, config.MaxConcurrent*2)
	downloadedAssetsCh := make(chan internal.Queable, config.MaxConcurrent*2)
	parsedPagesCh := make(chan internal.Queable, config.MaxConcurrent*2)
	saveCh := make(chan internal.Queable, config.MaxConcurrent*2)
	resultsCh := make(chan internal.Queable)

	// pools
	var httpPool = &sync.Pool{New: func() any { return httpclient.NewClient(httpclient.WithTimeout(config.Timeout)) }}

	// start workers
	startDownloadWorkers(g, ctx, config.MaxConcurrent, pageDownloadCh, downloadedPagesCh, httpPool, logger, true)
	startDownloadWorkers(g, ctx, config.MaxConcurrent, assetDownloadCh, downloadedAssetsCh, httpPool, logger, false)

	startParseWorkers(g, ctx, config.MaxConcurrent, downloadedPagesCh, parsedPagesCh, func(child internal.Queable) {
		// children: pages -> enqueue to pageDownloadCh; assets -> assetDownloadCh
		if _, ok := child.(internal.Parsable); ok {
			select {
			case <-ctx.Done():
				return
			case pageDownloadCh <- child:
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case assetDownloadCh <- child:
			}
		}
	}, logger)

	startSaveWorkers(g, ctx, config.MaxConcurrent, parsedPagesCh, downloadedAssetsCh, saveCh, logger)
	startPersistWorkers(g, ctx, config.MaxConcurrent, saveCh, resultsCh, config.OutputDir, logger)

	// feed start page
	startPage, err := internal.NewPage(config.StartURL)
	if err != nil {
		logger.Error("Failed to parse startURL", "err", err, "value", config.StartURL)
		os.Exit(1)
	}

	// seed
	select {
	case <-ctx.Done():
		logger.Warn("context done before seed")
	default:
		pageDownloadCh <- startPage
	}

	// wait for errgroup in background, and when it finishes close resultsCh
	g.Go(func() error {
		defer close(resultsCh)
		return g.Wait()
	})

	// consume results
	startedAt := time.Now()
	pagesCnt, assetsCnt := 0, 0

	for item := range resultsCh {
		switch item.(type) {
		case *internal.Page:
			pagesCnt++
			logger.Info(fmt.Sprintf("Done for page %d of %d", pagesCnt, config.MaxCount))
		default:
			assetsCnt++
			logger.Info(fmt.Sprintf("Done for asset '%s'", item.ItemId()))
		}
		// stop criteria: достигли лимита страниц
		if pagesCnt >= config.MaxCount {
			// аккуратно отменяем
			stop()
			break
		}
	}

	logger.Info("Crawling completed",
		"elapsed", time.Since(startedAt).String(),
		"pages_crawled", pagesCnt, "assets_crawled", assetsCnt,
	)
}

// signalWithCancel устанавливает контекст, отменяемый при SIGINT/SIGTERM
func signalWithCancel() (context.Context, func()) {
	ctx, stop := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case <-c:
			stop()
		}
	}()
	return ctx, stop
}

// worker starters -------------------------------------------------------------

func startDownloadWorkers(g *errgroup.Group, ctx context.Context, workers int, in <-chan internal.Queable, out chan<- internal.Queable, httpPool *sync.Pool, logger *slog.Logger, expectParsible bool) {
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case item, ok := <-in:
					if !ok {
						return nil
					}

					// Optional: skip downloading pages that are not expected here
					if expectParsible {
						// pages expected
					} else {
						// assets expected
					}

					logId := item.ItemId()
					logger.Debug(fmt.Sprintf("[%d] download recv %s", i, logId))

					downloadable := item.(internal.Downloadable)
					_, err := retry.Retry[int](ctx, func() (int, error) {
						dErr := downloadItem(ctx, downloadable, httpPool)
						if dErr != nil {
							return 0, dErr
						}
						return downloadable.GetSize(), nil
					}, retry.NewConfig(retry.WithMaxAttempts(3), retry.WithDelay(time.Second)))
					if err != nil {
						logger.Debug(fmt.Sprintf("download failed %s: %v", logId, err))
						continue
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					case out <- item:
					}
				}
			}
		})
	}
}

func startParseWorkers(g *errgroup.Group, ctx context.Context, workers int, in <-chan internal.Queable, out chan<- internal.Queable, pushChild func(internal.Queable), logger *slog.Logger) {
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case item, ok := <-in:
					if !ok {
						return nil
					}

					logId := item.ItemId()
					logger.Debug(fmt.Sprintf("parse recv %s", logId))

					parsable, ok := item.(internal.Parsable)
					if !ok {
						// asset passthrough
						select {
						case <-ctx.Done():
							return ctx.Err()
						case out <- item:
						}
						continue
					}

					if err := parsable.Parse(); err != nil {
						logger.Debug(fmt.Sprintf("parse error %s: %v", logId, err))
						// even on parse error we still pass item to save so we don't lose content
						select {
						case <-ctx.Done():
							return ctx.Err()
						case out <- item:
						}
						continue
					}

					for _, child := range parsable.GetChildren() {
						pushChild(child)
					}

					select {
					case <-ctx.Done():
						return ctx.Err()
					case out <- item:
					}
				}
			}
		})
	}
}

func startSaveWorkers(g *errgroup.Group, ctx context.Context, workers int, pagesIn <-chan internal.Queable, assetsIn <-chan internal.Queable, out chan<- internal.Queable, logger *slog.Logger) {
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case item, ok := <-pagesIn:
					if ok {
						// page save
						if sv, ok := item.(internal.Savable); ok {
							p, err := saveItem(ctx, "./out", sv)
							if err != nil {
								logger.Debug(fmt.Sprintf("save page failed %s: %v", item.ItemId(), err))
								continue
							}
							logger.Debug(fmt.Sprintf("saved page %s -> %s", item.ItemId(), p))
							select {
							case <-ctx.Done():
								return ctx.Err()
							case out <- item:
							}
						}
						continue
					}

				case asset, ok := <-assetsIn:
					if ok {
						if sv, ok := asset.(internal.Savable); ok {
							p, err := saveItem(ctx, "./out", sv)
							if err != nil {
								logger.Debug(fmt.Sprintf("save asset failed %s: %v", asset.ItemId(), err))
								continue
							}
							logger.Debug(fmt.Sprintf("saved asset %s -> %s", asset.ItemId(), p))
							select {
							case <-ctx.Done():
								return ctx.Err()
							case out <- asset:
							}
						}
						continue
					}
				}
			}
		})
	}
}

func startPersistWorkers(g *errgroup.Group, ctx context.Context, workers int, in <-chan internal.Queable, results chan<- internal.Queable, outDir string, logger *slog.Logger) {
	for i := 0; i < workers; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case item, ok := <-in:
					if !ok {
						return nil
					}
					// final persist already handled in save workers; just forward to results
					select {
					case <-ctx.Done():
						return ctx.Err()
					case results <- item:
					}
				}
			}
		})
	}
}

// downloadItem and saveItem остаются как у тебя, но с небольшими корректировками
func downloadItem(ctx context.Context, item internal.Downloadable, httpClientPool *sync.Pool) error {
	httpClient := httpClientPool.Get().(*httpclient.Client)
	defer httpClientPool.Put(httpClient)

	content, err := httpClient.Get(ctx, item.GetURL())
	if err != nil {
		return err
	}

	if err := item.SetContent(content); err != nil {
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
