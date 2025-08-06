package crawler

func sliceMap[T any, R any](slice []T, fn func(T) R) []R {
	res := make([]R, len(slice))
	for i, v := range slice {
		res[i] = fn(v)
	}
	return res
}

type Semaphore chan struct{}

func NewSemaphore(n int) Semaphore {
	return make(chan struct{}, n)
}

func (s Semaphore) Acquire() {
	s <- struct{}{} //записываем, и заблокируем при достижении лимитов
}

func (s Semaphore) Release() {
	<-s // вычитываем
}
