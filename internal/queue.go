package internal

import (
	"log/slog"
	"sync"
)

type Queue struct {
	pages        map[string]Queable
	assets       map[string]Queable
	seen         map[string]struct{}
	outCh        chan Queable
	mu           sync.Mutex
	logger       *slog.Logger
	pagesLimit   int
	pagesDoneCnt int
	once         sync.Once
}

func NewQueue(pagesLimit int, bufferSize int, logger *slog.Logger) *Queue {
	return &Queue{
		pages:      make(map[string]Queable),
		assets:     make(map[string]Queable),
		seen:       make(map[string]struct{}),
		outCh:      make(chan Queable, bufferSize),
		logger:     logger,
		pagesLimit: pagesLimit,
	}
}

func (q *Queue) Out() <-chan Queable {
	return q.outCh
}

func (q *Queue) Push(d Queable) bool {
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
		// limit exceeded
		if q.pagesDoneCnt >= q.pagesLimit {
			return false
		}

		q.pages[itemId] = d
		q.pagesDoneCnt++
	default:
		q.assets[itemId] = d
	}

	q.seen[itemId] = struct{}{}

	// blocks pushing, should we push to buffer and use goroutine to write to channel?
	q.outCh <- d

	// fmt.Println("out", len(q.outCh), "pages", q.pagesDoneCnt)

	return true
}

func (q *Queue) Ack(d Queable) {
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
		q.once.Do(func() {
			q.logger.Debug("Pages queue is empty")
			close(q.outCh)
		})
	}
}
