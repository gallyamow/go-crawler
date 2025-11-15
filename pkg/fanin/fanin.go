package fanin

import "sync"

// Merge простая реализация FanIn паттерна (специально без поддержки контекста).
func Merge[T any](inputChs ...<-chan T) <-chan T {
	out := make(chan T)

	var wg sync.WaitGroup
	wg.Add(len(inputChs))

	for _, inputCh := range inputChs {
		go func() {
			defer wg.Done()

			for v := range inputCh {
				out <- v
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
