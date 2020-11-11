package socket

import (
	"context"
	"sync"
)

type Scheduler struct {
	Conn      *Connection
	Ctx       context.Context
	WriteChan chan<- []byte

	rw     sync.RWMutex
	values map[interface{}]interface{}
}

func (sc *Scheduler) Send(b []byte) {
	sc.WriteChan <- b
}

func (sc *Scheduler) Store(key interface{}, value interface{}) {
	sc.rw.Lock()
	sc.values[key] = value
	sc.rw.Unlock()
}

func (sc *Scheduler) Load(key interface{}) (interface{}, bool) {
	sc.rw.RLock()
	defer sc.rw.RUnlock()
	if value, ok := sc.values[key]; ok {
		return value, true
	}
	return nil, false
}

func (sc *Scheduler) Delete(key interface{}) {
	sc.rw.Lock()
	delete(sc.values, key)
	sc.rw.Unlock()
}

func (sc *Scheduler) LoadAndDelete(key interface{}) (interface{}, bool) {
	sc.rw.Lock()
	defer sc.rw.Unlock()
	if value, ok := sc.values[key]; ok {
		delete(sc.values, key)
		return value, true
	}
	return nil, false
}

func (sc *Scheduler) Increment(key interface{}) int {
	sc.rw.Lock()
	defer sc.rw.Unlock()
	var cnt int
	if tmp, ok := sc.values[key]; ok {
		cnt = tmp.(int)
		cnt++
	} else {
		cnt = 0
	}
	sc.values[key] = cnt
	return cnt
}

func (sc *Scheduler) Decrement(key interface{}) int {
	sc.rw.Lock()
	defer sc.rw.Unlock()
	var cnt int
	if tmp, ok := sc.values[key]; ok {
		cnt = tmp.(int)
		cnt--
	} else {
		cnt = 0
	}
	sc.values[key] = cnt
	return cnt
}

func NewScheduler(ctx context.Context, connection *Connection, ch chan<- []byte) *Scheduler {
	return &Scheduler{
		Conn:      connection,
		Ctx:       ctx,
		WriteChan: ch,
		values:    make(map[interface{}]interface{}),
	}
}
