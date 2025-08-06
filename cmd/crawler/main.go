package main

import (
	"go-crawler/internal/crawler"
	"log/slog"
	"os"
	"sync"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	maxCount := 10
	maxConcurrent := 1
	startUrl := "https://go.dev/learn/"

	fetcher := crawler.NewFetcher()
	saver := crawler.NewSaver("./.tmp/")
	parser := crawler.NewParser()

	var handler func(queuedUrl string) (*crawler.ParseResult, *crawler.SaveResult, error)
	handler = func(queuedUrl string) (*crawler.ParseResult, *crawler.SaveResult, error) {
		page, err := fetcher.FetchPage(queuedUrl)
		if err != nil {
			return nil, nil, err
		}

		parsed, err := parser.Parse(page)
		if err != nil {
			return nil, nil, err
		}

		// TODO: change content before saving (urls, assets)

		saved, err := saver.SavePage(page)
		if err != nil {
			return nil, nil, err
		}

		return parsed, saved, nil
	}

	startedAt := time.Now()
	sem := crawler.NewSemaphore(maxConcurrent)
	wg := sync.WaitGroup{}
	queue := []string{startUrl}
	cnt := 0

	for len(queue) > 0 {
		url := queue[0]
		queue = queue[1:]

		wg.Add(1)
		go func() {
			defer wg.Done()

			sem.Acquire()
			defer sem.Release()

			parsed, saved, err := handler(url)
			if err != nil {
				logger.Error("Failed to handle", "err", err, "url", url)
				return
			}

			logger.Info("Successfully handled", "url", url, "parsed", parsed.String(), "saved", saved.String())
			cnt += 1

			// TODO: обработать дубли
			queue = append(queue, parsed.Links...)
		}()

		if cnt >= maxCount {
			logger.Info("Limit exceed", "limit", maxCount)
			break
		}

		wg.Wait()
	}

	wg.Wait()

	logger.Info("Elapsed time", "time", time.Since(startedAt).String())
}
