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

type Fetcher struct {
	client    *http.Client
	userAgent string
}

type OptionFunc func(*Fetcher)

func NewFetcher(options ...OptionFunc) *Fetcher {
	f := &Fetcher{
		client:    &http.Client{Timeout: defaultTimeout},
		userAgent: defaultUserAgent,
	}

	for _, opt := range options {
		opt(f)
	}

	return f
}

func FetcherWithTimeout(timeout time.Duration) OptionFunc {
	return func(f *Fetcher) {
		f.client.Timeout = timeout
	}
}

func FetcherWithUserAgent(ua string) OptionFunc {
	return func(f *Fetcher) {
		f.userAgent = ua
	}
}

func (f *Fetcher) GetPage(url string) (*Page, error) {
	content, err := f.GetURL(url)
	if err != nil {
		return nil, err
	}

	return &Page{
		Content: content,
		URL:     url,
	}, nil
}

func (f *Fetcher) GetURL(url string) ([]byte, error) {
	req, err := f.buildGetRequest(url)
	if err != nil {
		return nil, fmt.Errorf("build request failed: %w", err)
	}
	fmt.Println(req)

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexceprted status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (f *Fetcher) buildGetRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to init request: %w", err)
	}
	req.Header.Set("User-Agent", f.userAgent)

	return req, nil
}
