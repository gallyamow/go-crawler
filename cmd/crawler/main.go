package main

import (
	"go-crawler/internal/crawler"
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	maxCount := 10
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

	queue := []string{startUrl}
	cnt := 0

	for len(queue) > 0 {
		url := queue[0]
		queue = queue[1:]

		logger.Info("Handling url", "url", url)

		parsed, saved, err := handler(url)
		if err != nil {
			logger.Error("Failed to handle", "err", err, "url", url)
			continue
		}

		logger.Info("Successfully handled", "url", url, "parsed", parsed.String(), "saved", saved.String())
		cnt += 1

		if cnt >= maxCount {
			logger.Info("Limit exceed", "limit", maxCount)
			break
		}

		// TODO: обработать дубли
		queue = append(queue, parsed.Links...)
	}
}
