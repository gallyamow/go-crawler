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
	maxConcurrent := 5
	startUrl := "https://go.dev/learn/"

	fetcher := crawler.NewFetcher()
	saver := crawler.NewSaver("./.tmp/")
	parser := crawler.NewParser()

	var handler func(queuedUrl string) (*crawler.ParseResult, *crawler.SaveResult, error)
	handler = func(pageURL string) (*crawler.ParseResult, *crawler.SaveResult, error) {
		page, err := fetcher.FetchPage(pageURL)
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

	queue := []string{startUrl}
	visited := make(map[string]struct{})
	cnt := 0

	wg := sync.WaitGroup{}
	sem := crawler.NewSemaphore(maxConcurrent)

	for len(queue) > 0 {
		wg.Add(1)
		go func(queuedURL string) {
			sem.Acquire()

			defer wg.Done()
			defer sem.Release()

			parsed, saved, err := handler(queuedURL)
			if err != nil {
				logger.Error("Failed to handle", "err", err, "url", queuedURL)
				return
			}

			logger.Info("Successfully handled", "url", queuedURL, "parsed", parsed.String(), "saved", saved.String())
			cnt += 1

			if cnt >= maxCount {
				logger.Info("Page limit exceed", "limit", maxCount)
				return
			}

			for _, link := range parsed.Links {
				if _, ok := visited[link]; ok {
					continue
				}

				visited[link] = struct{}{}
				queue = append(queue, link)
			}
		}(queue[0])
		queue = queue[1:]
	}

	wg.Wait()
	logger.Info("Elapsed time", "time", time.Since(startedAt).String())
}
