package common

type Concurrentizer struct {
	limit int
}

type Waiter struct {
	done chan bool
}

func NewConcurrentizer(limit int) *Concurrentizer {
	return &Concurrentizer{limit}
}

func (c *Concurrentizer) Do(size int, ccf func()) *Waiter {
	if size < c.limit {
		ccf()
		return nil
	}
	w := &Waiter{make(chan bool, 1)}
	go func() {
		ccf()
		w.done <- true
	}()
	return w
}

func (w *Waiter) Wait() {
	if w != nil {
		<- w.done
	}
}
