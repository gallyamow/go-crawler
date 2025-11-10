package internal

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	MaxCount      int
	MaxConcurrent int
	StartURL      string
	Timeout       time.Duration
	RetryAttempts int
	RetryDelay    time.Duration
	OutputDir     string
	LogLevel      string
}

// LoadConfig loads configuration from environment variables and command line flags.
func LoadConfig() (*Config, error) {
	config := &Config{}

	// Set defaults
	config.MaxCount = getEnvInt("CRAWLER_MAX_COUNT", 100)
	config.MaxConcurrent = getEnvInt("CRAWLER_MAX_CONCURRENT", 10)
	config.StartURL = getEnvString("CRAWLER_START_URL", "https://go.dev/learn/")
	config.Timeout = getEnvDuration("CRAWLER_TIMEOUT", 30*time.Second)
	config.RetryAttempts = getEnvInt("CRAWLER_RETRY_ATTEMPTS", 3)
	config.RetryDelay = getEnvDuration("CRAWLER_RETRY_DELAY", 1*time.Second)
	config.OutputDir = getEnvString("CRAWLER_OUTPUT_DIR", "./.tmp/")
	config.LogLevel = getEnvString("CRAWLER_LOG_LEVEL", "info")

	// Parse command line flags
	flag.IntVar(&config.MaxCount, "max-count", config.MaxCount, "Maximum number of pages to crawl")
	flag.IntVar(&config.MaxConcurrent, "max-concurrent", config.MaxConcurrent, "Maximum number of concurrent workers")
	flag.StringVar(&config.StartURL, "start-url", config.StartURL, "Starting sourceURL for crawling")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "HTTP request timeout")
	flag.IntVar(&config.RetryAttempts, "retry-attempts", config.RetryAttempts, "Number of retry attempts for failed requests")
	flag.DurationVar(&config.RetryDelay, "retry-delay", config.RetryDelay, "Delay between retry attempts")
	flag.StringVar(&config.OutputDir, "output-dir", config.OutputDir, "Directory to save crawled pages")
	flag.StringVar(&config.LogLevel, "log-level", config.LogLevel, "Log level (debug, info, warn, error)")

	flag.Parse()

	// Validate configuration
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if c.MaxCount <= 0 {
		return fmt.Errorf("max-count must be positive, got %d", c.MaxCount)
	}
	if c.MaxConcurrent <= 0 {
		return fmt.Errorf("max-concurrent must be positive, got %d", c.MaxConcurrent)
	}
	if c.StartURL == "" {
		return fmt.Errorf("start-url cannot be empty")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", c.Timeout)
	}
	if c.RetryAttempts < 0 {
		return fmt.Errorf("retry-attempts cannot be negative, got %d", c.RetryAttempts)
	}
	if c.RetryDelay < 0 {
		return fmt.Errorf("retry-delay cannot be negative, got %v", c.RetryDelay)
	}
	if c.OutputDir == "" {
		return fmt.Errorf("output-dir cannot be empty")
	}

	return nil
}

func (c *Config) String() string {
	return fmt.Sprintf(
		"Config{MaxCount: %d, MaxConcurrent: %d, StartURL: %s, Timeout: %v, RetryAttempts: %d, RetryDelay: %v, OutputDir: %s, LogLevel: %s}",
		c.MaxCount, c.MaxConcurrent, c.StartURL, c.Timeout, c.RetryAttempts, c.RetryDelay, c.OutputDir, c.LogLevel,
	)
}

func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
