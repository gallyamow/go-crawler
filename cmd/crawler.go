package main

import (
	"fmt"
	"github.com/gallyamow/go-crawler/internal/crawler"
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	maxCount := 100
	maxConcurrent := 10
	startUrl := "https://go.dev/learn/"

	startedAt := time.Now()

	visited := make(map[string]struct{})
	cnt := 0

	fetcher := crawler.NewFetcher()
	saver := crawler.NewSaver("./.tmp/")
	parser := crawler.NewParser()

	handleUrl := func(url string) (*crawler.ParseResult, *crawler.SaveResult, error) {
		page, err := fetcher.FetchPage(url)
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

	worker := func(i int, jobs <-chan string, results chan<- *crawler.ParseResult) {
		logger.Info(fmt.Sprintf("Worker %d started", i))

		for url := range jobs {
			logger.Info(fmt.Sprintf("Worker %d is handling %s", i, url))

			parsed, saved, err := handleUrl(url)

			if err != nil {
				logger.Error("Failed to handle", "err", err, "url", url)
				continue
			}
			logger.Info("Successfully handled", "url", url, "parsed", parsed.String(), "saved", saved.String())

			results <- parsed
		}
	}

	jobs := make(chan string, maxConcurrent)
	results := make(chan *crawler.ParseResult, maxConcurrent)

	for i := range maxConcurrent {
		go worker(i, jobs, results)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for parsed := range results {
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
				if cnt < maxCount {
					select {
					case jobs <- link:
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
