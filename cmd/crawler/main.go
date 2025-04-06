package main

import (
	"fmt"
	"go-crawler/internal/crawler"
	"log"
)

func main() {
	url := "https://go.dev/learn/"

	fetcher := crawler.NewFetcher()
	page, err := fetcher.GetPage(url)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}

	saver := crawler.NewSaver("~/")
	saveResult, err := saver.SavePage(page)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}

	fmt.Println(saveResult)
}
