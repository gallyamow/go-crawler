package internal

import (
	"container/list"
	"context"
	"log/slog"
	"sync"
)

type Queue struct {
	pending          *list.List
	seen             map[string]struct{}
	pagesCh          chan Queueable
	assetsCh         chan Queueable
	mu               sync.Mutex
	cond             *sync.Cond
	logger           *slog.Logger
	pagesLimit       int
	totalQueuedPages int
	pendingAckCount  int
}

func NewQueue(ctx context.Context, pagesLimit int, chanSize int, logger *slog.Logger) *Queue {
	queue := &Queue{
		pending:    list.New(),
		seen:       make(map[string]struct{}),
		pagesCh:    make(chan Queueable, chanSize),
		assetsCh:   make(chan Queueable, chanSize),
		logger:     logger,
		pagesLimit: pagesLimit,
	}

	queue.cond = sync.NewCond(&queue.mu)

	go func(ctx context.Context, q *Queue) {
		for {
			if ctx.Err() != nil {
				return
			}

			// @idiomatic: cond instead of busy-wait
			q.cond.L.Lock()
			if q.pending.Len() == 0 {
				q.cond.Wait()
			}
			q.cond.L.Unlock()

			// @idiomatic: using container/list
			for e := q.pending.Front(); e != nil; {
				if ctx.Err() != nil {
					return
				}

				next := e.Next()
				item := e.Value.(Queueable)

				ch := q.assetsCh
				if _, ok := item.(*Page); ok {
					ch = q.pagesCh
				}

				select {
				case <-ctx.Done():
					return
				case ch <- item:
					q.pending.Remove(e)
				}

				e = next
			}
		}
	}(ctx, queue)

	return queue
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
func (q *Queue) Push(item Queueable) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	itemId := item.ItemId()
	if _, ok := q.seen[itemId]; ok {
		return false
	}
	q.seen[itemId] = struct{}{}

	// @idiomatic: compile time type checking
	// var _ Downloadable = (*CssFile)(nil)

	if _, ok := item.(*Page); ok {
		if q.totalQueuedPages >= q.pagesLimit {
			return false
		}
		q.totalQueuedPages++
	}

	q.pending.PushBack(item)
	q.cond.Signal()

	q.pendingAckCount++

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
		close(q.pagesCh)
		close(q.assetsCh)
	}
}
