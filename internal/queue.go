package internal

import (
	"log/slog"
	"sync"
)

type Queue struct {
	seen             map[string]struct{}
	pagesCh          chan Queable
	assetsCh         chan Queable
	mu               sync.Mutex
	logger           *slog.Logger
	pagesLimit       int
	totalQueuedPages int
	pendingAckCount  int
	once             sync.Once
}

func NewQueue(pagesLimit int, bufferSize int, logger *slog.Logger) *Queue {
	q := &Queue{
		seen:       make(map[string]struct{}),
		pagesCh:    make(chan Queable, bufferSize),
		assetsCh:   make(chan Queable, bufferSize),
		logger:     logger,
		pagesLimit: pagesLimit,
	}

	return q
}

func (q *Queue) Pages() <-chan Queable {
	return q.pagesCh
}

func (q *Queue) Assets() <-chan Queable {
	return q.assetsCh
}

func (q *Queue) Push(item Queable) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	itemId := item.ItemId()
	if _, ok := q.seen[itemId]; ok {
		return false
	}

	// @idiomatic: compile time type checking
	// var _ Downloadable = (*CssFile)(nil)

	switch item.(type) {
	case *Page:
		// checking total limits
		if q.totalQueuedPages >= q.pagesLimit {
			return false
		}

		q.totalQueuedPages++
		q.pagesCh <- item
	default:
		// assets
		q.assetsCh <- item
	}

	q.pendingAckCount++
	q.seen[itemId] = struct{}{}

	return true
}

func (q *Queue) Ack(item Queable) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.pendingAckCount--

	// (it doesn't look like robust way)
	// (is it valid way to check if we should stop?)
	// (способ через отдельную writing-goroutine тоже сложен)
	if q.pendingAckCount == 0 {
		q.once.Do(func() {
			close(q.pagesCh)
		})
	}
}
