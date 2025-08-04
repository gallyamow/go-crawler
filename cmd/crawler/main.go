package main

import (
	"fmt"
	"go-crawler/internal/crawler"
	"log"
)

func main() {
	url := "https://go.dev/learn/"

	fetcher := crawler.NewFetcher()
	page, err := fetcher.FetchPage(url)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}

	//saver := crawler.NewSaver("~/")
	saver := crawler.NewSaver("/Users/ramil/crawler-files/")
	saveResult, err := saver.SavePage(page)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
	fmt.Println(saveResult)

	parser := crawler.NewParser()
	parsed, err := parser.Parse(page)
	if err != nil {
		log.Fatalf("Fatal error: %v", err)
	}
	fmt.Println(parsed)
	fmt.Printf("%#v\n", parsed)
}
