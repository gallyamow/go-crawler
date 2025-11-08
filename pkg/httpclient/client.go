package httpclient

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Linux; Android 8.0.0; SM-G955U Build/R16NW)"
	defaultTimeout   = 30 * time.Second
)

type Client struct {
	client    *http.Client
	userAgent string
}

type OptionFunc func(*Client)

func NewClient(options ...OptionFunc) *Client {
	f := &Client{
		client:    &http.Client{Timeout: defaultTimeout},
		userAgent: defaultUserAgent,
	}

	for _, opt := range options {
		opt(f)
	}

	return f
}

func WithTimeout(timeout time.Duration) OptionFunc {
	return func(f *Client) {
		f.client.Timeout = timeout
	}
}

func WithUserAgent(ua string) OptionFunc {
	return func(f *Client) {
		f.userAgent = ua
	}
}

func (c *Client) Get(ctx context.Context, url string) ([]byte, error) {
	req, err := c.getRequest(url)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	content, err := c.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (c *Client) doRequest(ctx context.Context, req *http.Request) ([]byte, error) {
	req = req.WithContext(ctx)

	resp, err := c.client.Do(req)
	if err != nil {
		// @idiomatic: detect context-related errors
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			return nil, fmt.Errorf("failed to make http request: %w", err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) getRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to nit request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)

	return req, nil
}
