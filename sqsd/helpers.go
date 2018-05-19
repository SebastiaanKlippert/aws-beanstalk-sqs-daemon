package sqsd

import "sync"

type counter struct {
	value int
	sync.Mutex
}

func (c *counter) Get() int {
	c.Lock()
	defer c.Unlock()
	return c.value
}

func (c *counter) Add(n int) int {
	c.Lock()
	defer c.Unlock()
	c.value += n
	return c.value
}
