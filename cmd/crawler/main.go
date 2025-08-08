package main

import (
	"fmt"
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
	visited := make(map[string]struct{})
	queue := []string{startUrl}
	cnt := 0

	wg.Add(1) // ждем цикл
	for len(queue) > 0 {
		queuedURL := queue[0]
		queue = queue[1:]

		go func(u string) {
			fmt.Println("DDDDDDDDDDDDDDD" + u)
			defer wg.Done()

			sem.Acquire()
			defer sem.Release()

			parsed, saved, err := handler(u)
			if err != nil {
				logger.Error("Failed to handle", "err", err, "url", u)
				return
			}

			logger.Info("Successfully handled", "url", u, "parsed", parsed.String(), "saved", saved.String())
			cnt += 1

			for _, link := range parsed.Links {
				if _, ok := visited[link]; ok {
					continue
				}

				visited[link] = struct{}{}
				queue = append(queue, link)
			}
		}(queuedURL)

		if cnt >= maxCount {
			logger.Info("Page limit exceed", "limit", maxCount)
			break
		}

		// приходится ждать в цикле, без этого срабатывает только для одной записи
		wg.Wait()
	}

	logger.Info("Elapsed time", "time", time.Since(startedAt).String())
}
