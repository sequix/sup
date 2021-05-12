package util

import (
	"sync"
)

type BroadcastCh chan struct{}

func NewBroadcastCh() BroadcastCh { return make(chan struct{}) }
func (bc BroadcastCh) Broadcast() { close(bc) }
func (bc BroadcastCh) Wait()      { <-bc }

type Runner struct {
	stop BroadcastCh
	done *sync.WaitGroup
}

func (rw *Runner) Stop()        { rw.stop.Broadcast() }
func (rw *Runner) Wait()        { rw.done.Wait() }
func (rw *Runner) StopAndWait() { rw.Stop(); rw.Wait() }

type RunFunc func(BroadcastCh)

func Run(rfs ...RunFunc) *Runner {
	if len(rfs) == 0 {
		return nil
	}
	rw := &Runner{
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
