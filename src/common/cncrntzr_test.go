package common

import "testing"

func TestCon(t *testing.T) {
	c := NewConcurrentizer(2)
	count := 0
	c.Do(0, func() { 
		count++
	}).Wait()
	c.Do(1, func() { 
		count++
	}).Wait()
	c.Do(2, func() { 
		count++
	}).Wait()
	if count != 3 {
		t.Errorf("Invalid count: %v", count)
	}
	w1 := c.Do(1, func() { count++ })
	w2 := c.Do(1, func() { count++ })
	w3 := c.Do(1, func() { count++ })
	w1.Wait()
	w2.Wait()
	w3.Wait()
	if count != 6 {
		t.Errorf("Invalid count: %v", count)
	}
}
