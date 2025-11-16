## go-crawler

A minimalistic, concurrent web crawler written in Go.

## Features

- **Concurrent Processing**: Configurable number of worker goroutines
- **Graceful Shutdown**: Proper cleanup and signal handling
- **Retry Logic**: Exponential backoff with configurable retry attempts
- **Configuration Management**: Environment variables and command-line flags
- **Error Handling**: Error handling with detailed logging
- **Memory Management**: Efficient memory usage with proper cleanup

## Usage

```shell
go run cmd/crawler/main.go \
  --max-count 200 \
  --max-concurrent 20 \
  --url "https://go.dev/learn/" \
  --timeout 60s \
  --output-dir "./tmp"
```

```shell
./crawler --max-count 200 --max-concurrent 20 --url "https://go.dev/learn/" --output-dir "./.tmp"
```

## Options

| Flag               | Environment Variable     | Default | Description                |
|--------------------|--------------------------|---------|----------------------------|
| `--max-count`      | `CRAWLER_MAX_COUNT`      | 100     | Maximum pages to crawl     |
| `--max-concurrent` | `CRAWLER_MAX_CONCURRENT` | 10      | Maximum concurrent workers |
| `--url`            | `CRAWLER_URL`            | ""      | Starting URL               |
| `--timeout`        | `CRAWLER_TIMEOUT`        | 30s     | HTTP request timeout       |
| `--retry-attempts` | `CRAWLER_RETRY_ATTEMPTS` | 3       | Number of retry attempts   |
| `--retry-delay`    | `CRAWLER_RETRY_DELAY`    | 1s      | Delay between retries      |
| `--output-dir`     | `CRAWLER_OUTPUT_DIR`     | ./.tmp/ | Output directory           |
| `--log-level`      | `CRAWLER_LOG_LEVEL`      | info    | Log level                  |

## Future Enhancements

- [ ] Distributed crawling support
- [ ] Advanced filtering and crawling rules (by size, file format)
- [ ] Metrics & Monitoring: Comprehensive statistics and performance tracking

## To see

* https://github.com/gocolly/colly
* https://www.zenrows.com/blog/golang-web-crawler
* https://github.com/lizongying/go-crawler