package main

import (
	"fmt"
	"go-crawler/internal/crawler"
	"log/slog"
	"os"
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

	queue := make(chan string, maxConcurrent)
	visited := make(map[string]struct{})
	cnt := 0

	for qu := range queue {
		fmt.Println("DDDDDDDDDDDDDDD" + qu)
		go func(u string) {
			fmt.Println("DDDDDDDDDDDDDDD" + u)

			parsed, saved, err := handler(u)
			if err != nil {
				logger.Error("Failed to handle", "err", err, "url", u)
				return
			}

			logger.Info("Successfully handled", "url", u, "parsed", parsed.String(), "saved", saved.String())
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
				queue <- link
			}
		}(qu)
	}

	go func() {
		queue <- startUrl
	}()

	logger.Info("Elapsed time", "time", time.Since(startedAt).String())
}
