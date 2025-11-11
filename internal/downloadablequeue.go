package internal

import (
	"sync"
)

type DownloadableQueue struct {
	items map[string]Downloadable
	seen  map[string]interface{}
	mu    sync.Mutex
}

func NewQueue() *DownloadableQueue {
	return &DownloadableQueue{
		items: make(map[string]Downloadable),
		seen:  make(map[string]interface{}),
	}
}

func (q *DownloadableQueue) Push(d Downloadable) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	url := d.GetURL()
	if _, ok := q.seen[url]; ok {
		return false
	}

	q.items[url] = d
	return true
}

func (q *DownloadableQueue) Ack(d Downloadable) Downloadable {
	q.mu.Lock()
	defer q.mu.Unlock()

	url := d.GetURL()
	q.items[url] = d

	delete(q.items, url)

	return d
}

func (q *DownloadableQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.items)
}
