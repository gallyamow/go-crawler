package internal

import (
	"log/slog"
	"sync"
)

type Queue struct {
	pages      map[string]Identifiable
	assets     map[string]Identifiable
	seen       map[string]struct{}
	outCh      chan Identifiable
	mu         sync.Mutex
	logger     *slog.Logger
	pagesLimit int
}

func NewQueue(limit int, bufferSize int, logger *slog.Logger) *Queue {
	return &Queue{
		pages:      make(map[string]Identifiable),
		assets:     make(map[string]Identifiable),
		seen:       make(map[string]struct{}),
		outCh:      make(chan Identifiable, bufferSize),
		logger:     logger,
		pagesLimit: limit,
	}
}

func (q *Queue) Out() <-chan Identifiable {
	return q.outCh
}

func (q *Queue) Push(d Identifiable) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	itemId := d.ItemId()
	if _, ok := q.seen[itemId]; ok {
		return false
	}

	// @idiomatic: compile time type checking
	// var _ Downloadable = (*CssFile)(nil)

	switch d.(type) {
	case *Page:
		if q.pagesLimit <= 0 {
			return false
		}
		q.pages[itemId] = d
		q.pagesLimit--
	default:
		q.assets[itemId] = d
	}

	q.seen[itemId] = struct{}{}

	// blocks pushing, should we push to buffer and use goroutine to write to channel?
	q.outCh <- d

	return true
}

func (q *Queue) Ack(d Identifiable) {
	q.mu.Lock()
	defer q.mu.Unlock()

	itemId := d.ItemId()

	switch d.(type) {
	case *Page:
		delete(q.pages, itemId)
		// it doesn't look like robust way
		// (is it valid way to check if we should stop?)
	default:
		delete(q.assets, itemId)
	}

	// it doesn't look like robust way
	// (is it valid way to check if we should stop?)
	if len(q.pages) == 0 {
		var once sync.Once
		once.Do(func() {
			q.logger.Debug("Pages queue is empty")
			close(q.outCh)
		})
	}
}
