package internal

import (
	"sync"
)

type DownloadableQueue struct {
	pages  map[string]Downloadable
	assets map[string]Downloadable
	seen   map[string]interface{}
	mu     sync.Mutex
}

func NewQueue() *DownloadableQueue {
	return &DownloadableQueue{
		pages:  make(map[string]Downloadable),
		assets: make(map[string]Downloadable),
		seen:   make(map[string]interface{}),
	}
}

func (q *DownloadableQueue) Out() <-chan Downloadable {
	outCh := make(chan Downloadable)

	for _, d := range q.assets {
		outCh <- d
	}

	return outCh
}

func (q *DownloadableQueue) Push(d Downloadable) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	url := d.GetURL()
	if _, ok := q.seen[url]; ok {
		return false
	}

	var _ Downloadable = (*CssFile)(nil)

	switch d.(type) {
	case *Page:
		q.pages[url] = d
	default:
		q.assets[url] = d
	}

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

	return d
}

func (q *DownloadableQueue) IsFinished() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pages)
}
