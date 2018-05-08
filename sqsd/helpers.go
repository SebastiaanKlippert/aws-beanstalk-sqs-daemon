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

type signal struct {
	value bool
	sync.Mutex
}

func (s *signal) Get() bool {
	s.Lock()
	defer s.Unlock()
	return s.value
}

func (s *signal) Set(val bool) {
	s.Lock()
	defer s.Unlock()
	s.value = val
	return
}
