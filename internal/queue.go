package internal

import (
	"context"
	"log/slog"
	"sync"
)

type Queue struct {
	seen             map[string]struct{}
	pagesCh          chan Queueable
	assetsCh         chan Queueable
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
		pagesCh:    make(chan Queueable, bufferSize),
		assetsCh:   make(chan Queueable, bufferSize),
		logger:     logger,
		pagesLimit: pagesLimit,
	}

	return q
}

func (q *Queue) Pages() <-chan Queueable {
	return q.pagesCh
}

func (q *Queue) Assets() <-chan Queueable {
	return q.assetsCh
}

// Push помещает элемент в очередь на обработку.
// @idiomatic: deadlock due to holding a mutex while performing a potentially blocking operation
// (избавился от этой проблемы: использование здесь mutex приводит к тому что он остается захваченным до отправки в pagesCh или assetsCh)
func (q *Queue) Push(ctx context.Context, item Queueable) bool {
	if !q.commitAsSeen(item) {
		return false
	}

	if ctx.Err() != nil {
		return false
	}

	// @idiomatic: compile time type checking
	// var _ Downloadable = (*CssFile)(nil)

	if page, ok := item.(*Page); ok {
		// @idiomatic: early unlock
		q.mu.Lock()
		if q.totalQueuedPages >= q.pagesLimit {
			q.mu.Unlock()
			// total limits exceed
			return false
		}
		q.totalQueuedPages++
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			return false
		case q.pagesCh <- page:
		}
	} else {
		// assets
		q.assetsCh <- item
	}

	q.mu.Lock()
	q.pendingAckCount++
	q.mu.Unlock()

	return true
}

func (q *Queue) Ack(item Queueable) {
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

func (q *Queue) commitAsSeen(item Queueable) bool {
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
