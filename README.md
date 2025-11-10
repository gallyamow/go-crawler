## go-crawler

A minimalistic, concurrent web crawler written in Go.

## Features

- **Concurrent Processing**: Configurable number of worker goroutines
- **Graceful Shutdown**: Proper cleanup and signal handling
- **Retry Logic**: Exponential backoff with configurable retry attempts
- **Circuit Breaker**: Prevents cascading failures
- **Metrics & Monitoring**: Comprehensive statistics and performance tracking
- **Configuration Management**: Environment variables and command-line flags
- **Error Handling**: Robust error handling with detailed logging
- **Rate Limiting**: Built-in rate limiting to be respectful to servers

## Usage

```shell
go run cmd/crawler/main.go \
  --max-count 200 \
  --max-concurrent 20 \
  --start-url "https://example.com" \
  --timeout 60s \
  --output-dir "./output"
```

## Options

| Flag               | Environment Variable     | Default               | Description                |
|--------------------|--------------------------|-----------------------|----------------------------|
| `--max-count`      | `CRAWLER_MAX_COUNT`      | 100                   | Maximum pages to crawl     |
| `--max-concurrent` | `CRAWLER_MAX_CONCURRENT` | 10                    | Maximum concurrent workers |
| `--start-url`      | `CRAWLER_START_URL`      | https://go.dev/learn/ | Starting URL               |
| `--timeout`        | `CRAWLER_TIMEOUT`        | 30s                   | HTTP request timeout       |
| `--retry-attempts` | `CRAWLER_RETRY_ATTEMPTS` | 3                     | Number of retry attempts   |
| `--retry-delay`    | `CRAWLER_RETRY_DELAY`    | 1s                    | Delay between retries      |
| `--output-dir`     | `CRAWLER_OUTPUT_DIR`     | ./.tmp/               | Output directory           |
| `--log-level`      | `CRAWLER_LOG_LEVEL`      | info                  | Log level                  |

## Performance Features

- **Worker Pool**: Configurable number of concurrent workers
- **Channel Buffering**: Optimized channel sizes for better throughput
- **Memory Management**: Efficient memory usage with proper cleanup
- **Connection Pooling**: HTTP client reuse for better performance

## Monitoring & Observability

The crawler provides comprehensive metrics including:

- Crawl rate (pages per second)
- Success/failure rates
- Response time statistics
- Worker utilization
- Error tracking and categorization

## Error Handling

- **Network Errors**: Automatic retry with exponential backoff
- **Parse Errors**: Graceful handling of malformed HTML
- **Save Errors**: Continues processing even if saving fails
- **Rate Limiting**: Respects server limits and backs off

## Future Enhancements

- [ ] Distributed crawling support
- [ ] Advanced filtering and crawling rules (by size, file format)

## To see

* https://github.com/gocolly/colly
* https://www.zenrows.com/blog/golang-web-crawler
* https://github.com/lizongying/go-crawler