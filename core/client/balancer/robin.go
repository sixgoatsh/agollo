package balancer

import "sync/atomic"

type roundRobin struct {
	ss []string
	c  uint64
}

func NewRoundRobin(ss []string) Balancer {
	return &roundRobin{
		ss: ss,
		c:  0,
	}
}

func (rr *roundRobin) Select() (string, error) {
	if len(rr.ss) <= 0 {
		return "", ErrNoConfigServerAvailable
	}

	old := atomic.AddUint64(&rr.c, 1) - 1
	idx := old % uint64(len(rr.ss))
	return rr.ss[idx], nil
}

func (rr *roundRobin) Stop() {

}
