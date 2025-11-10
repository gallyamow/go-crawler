package internal

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Metrics struct {
	mu sync.RWMutex

	// counters
	PagesCrawled    int64
	PagesFailed     int64
	LinksDiscovered int64
	AssetsFound     int64
	BytesDownloaded int64

	// timing
	StartTime     time.Time
	LastCrawlTime time.Time

	// rate
	AverageResponseTime time.Duration
	TotalResponseTime   time.Duration
	ResponseCount       int64

	// error tracking
	LastError     string
	LastErrorTime time.Time
	ErrorCount    int64

	// worker statistics
	ActiveWorkers int64
	TotalWorkers  int64
}

func NewMetrics() *Metrics {
	return &Metrics{
		StartTime: time.Now(),
	}
}

func (m *Metrics) RecordPageCrawled() {
	atomic.AddInt64(&m.PagesCrawled, 1)
	m.mu.Lock()
	m.LastCrawlTime = time.Now()
	m.mu.Unlock()
}

func (m *Metrics) RecordPageFailed() {
	atomic.AddInt64(&m.PagesFailed, 1)
}

func (m *Metrics) RecordLinksDiscovered(count int) {
	atomic.AddInt64(&m.LinksDiscovered, int64(count))
}

func (m *Metrics) RecordAssetsFound(count int) {
	atomic.AddInt64(&m.AssetsFound, int64(count))
}

func (m *Metrics) RecordBytesDownloaded(bytes int) {
	atomic.AddInt64(&m.BytesDownloaded, int64(bytes))
}

func (m *Metrics) RecordResponseTime(duration time.Duration) {
	m.mu.Lock()
	m.TotalResponseTime += duration
	m.ResponseCount++
	m.AverageResponseTime = m.TotalResponseTime / time.Duration(m.ResponseCount)
	m.mu.Unlock()
}

func (m *Metrics) RecordError(err string) {
	m.mu.Lock()
	m.LastError = err
	m.LastErrorTime = time.Now()
	m.ErrorCount++
	m.mu.Unlock()
}

func (m *Metrics) SetActiveWorkers(count int) {
	atomic.StoreInt64(&m.ActiveWorkers, int64(count))
}

func (m *Metrics) SetTotalWorkers(count int) {
	atomic.StoreInt64(&m.TotalWorkers, int64(count))
}

func (m *Metrics) GetStats() Stats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return Stats{
		PagesCrawled:        atomic.LoadInt64(&m.PagesCrawled),
		PagesFailed:         atomic.LoadInt64(&m.PagesFailed),
		LinksDiscovered:     atomic.LoadInt64(&m.LinksDiscovered),
		AssetsFound:         atomic.LoadInt64(&m.AssetsFound),
		BytesDownloaded:     atomic.LoadInt64(&m.BytesDownloaded),
		StartTime:           m.StartTime,
		LastCrawlTime:       m.LastCrawlTime,
		AverageResponseTime: m.AverageResponseTime,
		ResponseCount:       atomic.LoadInt64(&m.ResponseCount),
		LastError:           m.LastError,
		LastErrorTime:       m.LastErrorTime,
		ErrorCount:          atomic.LoadInt64(&m.ErrorCount),
		ActiveWorkers:       atomic.LoadInt64(&m.ActiveWorkers),
		TotalWorkers:        atomic.LoadInt64(&m.TotalWorkers),
		Uptime:              time.Since(m.StartTime),
	}
}

// Snapshot for external usage
type Stats struct {
	PagesCrawled        int64
	PagesFailed         int64
	LinksDiscovered     int64
	AssetsFound         int64
	BytesDownloaded     int64
	StartTime           time.Time
	LastCrawlTime       time.Time
	AverageResponseTime time.Duration
	ResponseCount       int64
	LastError           string
	LastErrorTime       time.Time
	ErrorCount          int64
	ActiveWorkers       int64
	TotalWorkers        int64
	Uptime              time.Duration
}

func (s Stats) String() string {
	return fmt.Sprintf(
		"Pages: %d crawled, %d failed | Links: %d | asset: %d | Bytes: %d | Workers: %d/%d | Uptime: %v | Avg Response: %v",
		s.PagesCrawled, s.PagesFailed, s.LinksDiscovered, s.AssetsFound, s.BytesDownloaded,
		s.ActiveWorkers, s.TotalWorkers, s.Uptime, s.AverageResponseTime,
	)
}

func (s Stats) SuccessRate() float64 {
	total := s.PagesCrawled + s.PagesFailed
	if total == 0 {
		return 0
	}
	return float64(s.PagesCrawled) / float64(total) * 100
}

func (s Stats) CrawlRate() float64 {
	if s.Uptime.Seconds() == 0 {
		return 0
	}
	return float64(s.PagesCrawled) / s.Uptime.Seconds()
}
