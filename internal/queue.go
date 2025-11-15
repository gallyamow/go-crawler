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
	// @idiomatic: deadlock due to holding a mutex while performing a potentially blocking operation
	// (избавился от этой проблемы: использование здесь mutex приводит к тому что он остается захваченным до отправки в pagesCh или assetsCh)

	if !q.commitAsSeen(item) {
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

		q.mu.Lock()
		q.totalQueuedPages++
		q.mu.Unlock()

		q.pagesCh <- item
	default:
		// assets
		q.assetsCh <- item
	}

	q.mu.Lock()
	q.pendingAckCount++
	q.mu.Unlock()

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
		//q.once.Do(func() { // можно и без sync.
		close(q.pagesCh)
		close(q.assetsCh)
		//})
	}
}

func (q *Queue) commitAsSeen(item Queable) bool {
	// использование здесь mutex приводит к тому что он остается захваченным до отправки в pagesCh или assetsCh
	q.mu.Lock()
	defer q.mu.Unlock()

	itemId := item.ItemId()
	if _, ok := q.seen[itemId]; ok {
		return false
	}
	q.seen[itemId] = struct{}{}

	return true
}
