package common

import "log"
import "testing"
import "runtime"

func tree(n int, o chan<- bool, c *Concurrentizer) {
	if n == 0 {
		panic("What?")
	}
	if n == 1 {
		o <- true
		return
	}
	h := n / 2
	c.Do(h, func(ccon *Concurrentizer) {
		tree(h, o, ccon)
	})
	c.Do(n-h, func(ccon *Concurrentizer) {
		tree(n-h, o, ccon)
	})
	c.Wait()
}

func TestCon(t *testing.T) {
	runtime.GOMAXPROCS(runtime.NumCPU())
	const N = 100000
	c := NewConcurrentizer(10)
	o := make(chan bool, N+1)
	c.Do(N, func(ccon *Concurrentizer) {
		tree(N, o, ccon)
	}).Wait()
	close(o)
	count := 0
	for _ = range o {
		count++
		log.Println("Received ", count)
	}
	if count != N {
		t.Error("Failed: ", c)
	}
}
