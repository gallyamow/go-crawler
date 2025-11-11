package retry

import (
	"context"
	"math"
	"math/rand"
	"time"
)

// Retry - запускает функцию fn, в случае возврата не nil-error выполняет повторные ее запуски согласно конфигурации.
//
// Требования:
//   - принимает функцию для повторного выполнения
//   - принимает следующие конфигурационные параметры
//     MaxAttempts - максимальное кол-во попыток
//     Delay - задержка между вызовами
//     BackoffFactor - коэффициент увеличения задержки при каждом повторе (0 - если не увеличивать, больше 0 если требуется)
//     MaxDelay - максимально возможная задержка
//     JitterFactor - коэффициент до которого может случайным образом увеличиваться delay (0 - если не увеличивать, больше 0 если требуется)
//     RetryableChecker - функция принимающая ошибку и возвращающая true в случае если нужен повторный вызов, иначе false
//   - реагирует на отмену через контекст
//   - функция вызывается хотя бы 1 раз независимо от RetryableChecker
func Retry[T any](ctx context.Context, fn RetryableFunc[T], config *Config) (T, error) {
	if config == nil {
		config = DefaultConfig()
	}

	var zero T
	var lastErr error

	for attempt := range config.MaxAttempts {
		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		res, err := fn()
		if err == nil {
			return res, nil
		}

		lastErr = err

		if !config.RetryableChecker(err) {
			return zero, err
		}

		// @idiomatic: convert time.Duration to int64/float64 returns nanoseconds
		d := float64(config.Delay) * math.Pow(config.BackoffFactor, float64(attempt))
		d += d * rand.Float64() * config.JitterFactor

		// @idiomatic: type casting to time.Duration accepts nanoseconds
		delay := min(time.Duration(d), config.MaxDelay)

		select {
		case <-ctx.Done():
			return zero, ctx.Err()
		case <-time.After(delay):
			// waiting
		}
	}

	return zero, lastErr
}

// RetryableFunc запускаемая функция.
type RetryableFunc[T any] func() (T, error)

// RetryableCheckerFunc функция которая должна вернуть true в случае если необходимо повторить вызов
type RetryableCheckerFunc func(error) bool

type Config struct {
	MaxAttempts      int
	Delay            time.Duration
	MaxDelay         time.Duration
	BackoffFactor    float64
	JitterFactor     float64
	RetryableChecker RetryableCheckerFunc
}

// DefaultConfig возвращает конфигурацию по умолчанию.
// @idiomatic: Providing sensible defaults (to avoid using nil values)
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:   3,
		Delay:         1 * time.Second,
		MaxDelay:      3 * time.Second,
		BackoffFactor: 1,
		JitterFactor:  0.1,
		RetryableChecker: func(err error) bool {
			return true
		},
	}
}

func NewConfig(opts ...ConfigOptionFunc) *Config {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}

type ConfigOptionFunc func(*Config)

func WithMaxAttempts(val int) ConfigOptionFunc {
	return func(c *Config) {
		c.MaxAttempts = val
	}
}

func WithDelay(val time.Duration) ConfigOptionFunc {
	if val <= 0 {
		panic("delay must be greater than 0")
	}
	return func(c *Config) {
		c.Delay = val
	}
}

func WithMaxDelay(val time.Duration) ConfigOptionFunc {
	return func(c *Config) {
		c.MaxDelay = val
	}
}

func WithBackoffFactor(val float64) ConfigOptionFunc {
	if val < 0 {
		panic("backoff factor must be greater or equal than 0, pass 0 if you want to disable backoff")
	}

	return func(c *Config) {
		c.BackoffFactor = val
	}
}
func WithJitterFactor(val float64) ConfigOptionFunc {
	if val < 0 || val > 1 {
		panic("jitter must be in [0, 1], pass 0 if you want to disable jitter")
	}

	return func(c *Config) {
		c.JitterFactor = val
	}
}
func WithRetryableChecker(val RetryableCheckerFunc) ConfigOptionFunc {
	return func(c *Config) {
		c.RetryableChecker = val
	}
}
