package run

import (
	"sync"
)

type Runner struct {
	stop chan struct{}
	done *sync.WaitGroup
}

func (rw *Runner) Stop()        { close(rw.stop) }
func (rw *Runner) Wait()        { rw.done.Wait() }
func (rw *Runner) StopAndWait() { rw.Stop(); rw.Wait() }

type Func func(<-chan struct{})

func Run(rfs ...Func) *Runner {
	if len(rfs) == 0 {
		return nil
	}
	rw := &Runner{
		stop: make(chan struct{}),
		done: &sync.WaitGroup{},
	}
	rw.done.Add(len(rfs))
	for _, rf := range rfs {
		go func(rf Func) {
			rf(rw.stop)
			rw.done.Done()
		}(rf)
	}
	return rw
}
