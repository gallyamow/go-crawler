package crawler

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Linux; Android 8.0.0; SM-G955U Build/R16NW)"
	defaultTimeout   = 30 * time.Second
)

type Page struct {
	URL     string
	Content []byte
}

type Fetcher struct {
	client    *http.Client
	userAgent string
}

type FetcherOptionFunc func(*Fetcher)

func NewFetcher(options ...FetcherOptionFunc) *Fetcher {
	f := &Fetcher{
		client:    &http.Client{Timeout: defaultTimeout},
		userAgent: defaultUserAgent,
	}

	for _, opt := range options {
		opt(f)
	}

	return f
}

func FetcherWithTimeout(timeout time.Duration) FetcherOptionFunc {
	return func(f *Fetcher) {
		f.client.Timeout = timeout
	}
}

func FetcherWithUserAgent(ua string) FetcherOptionFunc {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

func (f *Fetcher) FetchPage(url string) (*Page, error) {
	req, err := f.buildGetRequest(url)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	content, err := f.doRequest(req)
	if err != nil {
		return nil, err
	}

	return &Page{
		Content: content,
		URL:     url,
	}, nil
}

func (f *Fetcher) doRequest(req *http.Request) ([]byte, error) {
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("make http-request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (f *Fetcher) buildGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("init request: %w", err)
	}
	req.Header.Set("User-Agent", f.userAgent)

	return req, nil
}
