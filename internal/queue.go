package internal

import (
	"sync"
)

type DownloadableQueue struct {
	pages  map[string]Downloadable
	assets map[string]Downloadable
	seen   map[string]struct{}
	outCh  chan Downloadable
	mu     sync.Mutex
}

func NewQueue(bufferSize int) *DownloadableQueue {
	return &DownloadableQueue{
		pages:  make(map[string]Downloadable),
		assets: make(map[string]Downloadable),
		seen:   make(map[string]struct{}),
		outCh:  make(chan Downloadable, bufferSize),
	}
}

func (q *DownloadableQueue) Out() <-chan Downloadable {
	return q.outCh
}

func (q *DownloadableQueue) Push(d Downloadable) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	url := d.GetURL()
	if _, ok := q.seen[url]; ok {
		return false
	}

	//var _ Downloadable = (*CssFile)(nil)

	switch d.(type) {
	case *Page:
		q.pages[url] = d
	default:
		q.assets[url] = d
	}

	q.seen[url] = struct{}{}

	q.outCh <- d

	return true
}

func (q *DownloadableQueue) Ack(d Downloadable) Downloadable {
	q.mu.Lock()
	defer q.mu.Unlock()

	url := d.GetURL()

	switch d.(type) {
	case *Page:
		delete(q.pages, url)
	default:
		delete(q.assets, url)
	}

	if len(q.pages) == 0 {
		close(q.outCh)
	}

	return d
}
