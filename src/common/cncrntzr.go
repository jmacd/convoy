package common

import "runtime"

type Concurrentizer struct {
	limit int          // Use-a-goroutine threshold 
	await int          // How many completions to wait for
	running chan bool  // A semaphore of for N (#CPUs) concurrent tasks
	waiting chan bool  // Communicate completion at one level
}

func NewConcurrentizer(limit int) *Concurrentizer {
	running := make(chan bool, runtime.NumCPU())
	running <- true
	return &Concurrentizer{limit, 0, running, make(chan bool, 2)}
}

func (c *Concurrentizer) Do(size int, 
	ccf func (*Concurrentizer)) *Concurrentizer {

	if size < c.limit {
		ccf(c)
		return c
	}

	c.await++
	go func(num int) {
		if num != 1 {
			c.running <- true
		}
		ccon := &Concurrentizer{c.limit, 0, c.running, 
			make(chan bool, 2)}
		ccf(ccon)
		c.waiting <- true
	}(c.await)
	return c
}

func (c *Concurrentizer) Wait() {
	for i := 0; i < c.await; i++ {
		if i != 0 {
			<- c.running
		}				
		<- c.waiting
	}
}
