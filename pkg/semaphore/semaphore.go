package semaphore

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
