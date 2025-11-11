package retry

import (
	"errors"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	t.Run("successful_call_should_not_retry", func(t *testing.T) {
		var retries int

		val, err := Retry[int](t.Context(), func() (int, error) {
			retries++
			return 1, nil
		}, nil)

		if err != nil {
			t.Fatalf("got error %v", err)
		}

		if val != 1 {
			t.Fatalf("got %v, want 1", val)
		}

		if retries != 1 {
			t.Fatalf("got retries")
		}
	})

	t.Run("unsuccessful_call_should_retry", func(t *testing.T) {
		var retries int

		config := NewConfig(WithMaxAttempts(3), WithMaxDelay(100*time.Millisecond))

		val, err := Retry[int](t.Context(), func() (int, error) {
			retries++
			return 0, errors.New("some error")
		}, config)

		if err == nil {
			t.Fatalf("want error")
		}

		if val == 1 {
			t.Fatalf("got %v, want zero value", val)
		}

		if retries != 3 {
			t.Fatalf("got %v retries, want 3", retries)
		}
	})

	t.Run("if_not_checked_should_not_retry", func(t *testing.T) {
		var retries int
		var checked int

		config := NewConfig(WithRetryableChecker(func(err error) bool {
			checked++
			return false
		}))

		val, err := Retry[int](t.Context(), func() (int, error) {
			retries++
			return 0, errors.New("some error")
		}, config)

		if err == nil {
			t.Fatalf("want error")
		}

		if val == 1 {
			t.Fatalf("got %v, want zero value", val)
		}

		if retries != 1 {
			t.Fatalf("got %v retries, want 1", retries)
		}

		if checked != 1 {
			t.Fatalf("got %v checked, want 1", checked)
		}
	})

	t.Run("jitter", func(t *testing.T) {
		var retries int
		var elapsed time.Duration
		var d = 100
		var jitter = 0.3

		config := NewConfig(
			WithBackoffFactor(0),
			WithJitterFactor(jitter),
			WithDelay(time.Duration(d)*time.Millisecond),
		)

		last := time.Now()
		val, err := Retry[int](t.Context(), func() (int, error) {
			retries++
			elapsed += time.Since(last)
			return 0, errors.New("some error")
		}, config)

		if err == nil {
			t.Fatalf("want error")
		}

		if val == 1 {
			t.Fatalf("got %v, want zero value", val)
		}

		if retries != 3 {
			t.Fatalf("got %v retries, want 3", retries)
		}

		// no backoff, only jitter
		maxElapsed := time.Duration(float64(retries*d)+float64(retries*d)*jitter) * time.Millisecond
		if elapsed > maxElapsed {
			t.Fatalf("elapsed %v, want <= %v", elapsed, maxElapsed)
		}
	})

	t.Run("backoff", func(t *testing.T) {
		var retries int
		var elapsed time.Duration
		var d = 100
		var backoff = 3.0

		config := NewConfig(
			WithBackoffFactor(backoff),
			WithJitterFactor(0),
			WithDelay(time.Duration(d)*time.Millisecond),
		)

		last := time.Now()
		val, err := Retry[int](t.Context(), func() (int, error) {
			retries++
			elapsed += time.Since(last)
			return 0, errors.New("some error")
		}, config)

		if err == nil {
			t.Fatalf("want error")
		}

		if val == 1 {
			t.Fatalf("got %v, want zero value", val)
		}

		if retries != 3 {
			t.Fatalf("got %v retries, want 3", retries)
		}

		// no jitter, only backoff
		maxElapsed := time.Duration(float64(d)+float64(d)*backoff+float64(d)*backoff*backoff) * time.Millisecond
		if elapsed > maxElapsed {
			t.Fatalf("elapsed %v, want <= %v", elapsed, maxElapsed)
		}
	})
}
