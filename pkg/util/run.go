package util

import (
	"sync"
)

type BroadcastCh chan struct{}

func NewBroadcastCh() BroadcastCh { return make(chan struct{}) }
func (bc BroadcastCh) Broadcast() { close(bc) }
func (bc BroadcastCh) Wait()      { <-bc }

type RunWrapper struct {
	stop BroadcastCh
	done *sync.WaitGroup
}

func (rw *RunWrapper) Stop() { rw.stop.Broadcast() }
func (rw *RunWrapper) Wait() { rw.done.Wait() }

func (rw *RunWrapper) StopAndWait() {
	rw.Stop()
	rw.Wait()
}

type RunFunc func(BroadcastCh)

func Run(rfs ...RunFunc) *RunWrapper {
	if len(rfs) == 0 {
		return nil
	}
	rw := &RunWrapper{
		stop: NewBroadcastCh(),
		done: &sync.WaitGroup{},
	}
	rw.done.Add(len(rfs))
	for _, rf := range rfs {
		go func(rf RunFunc) {
			rf(rw.stop)
			rw.done.Done()
		}(rf)
	}
	return rw
}
