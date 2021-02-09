package process

import (
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/procfs"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/util"
)

// TODO: config generate, validate, default, comment

type Controller struct {
	cmd          *exec.Cmd
	logWritePipe *io.PipeWriter
	logReadPipe  *io.PipeReader
	logger       *lumberjack.Logger
	waiterCh     chan struct{}
	wantStop     int32
}

func (c *Controller) Start(_ *Request, _ *Response) error {
	log.Info("recv start action")
	err := c.handleStart()
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) Stop(_ *Request, _ *Response) error {
	log.Info("recv stop action")
	err := c.handleStop()
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) Restart(_ *Request, _ *Response) error {
	log.Info("recv restart action")
	err := c.handleRestart()
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) Reload(_ *Request, _ *Response) error {
	log.Info("recv reload action")
	err := c.handleReload()
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) Kill(_ *Request, _ *Response) error {
	log.Info("recv kill action")
	err := c.handleKill()
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) Status(_ *Request, rsp *Response) error {
	log.Info("recv status action")
	err := c.handleStatus(rsp)
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) waiter(stop util.BroadcastCh) {
	go func() {
		<-stop
		c.handleStop()
		close(c.waiterCh)
	}()

	for {
		_, closed := <-c.waiterCh
		stat, err := c.cmd.Process.Wait()
		if err != nil {
			log.Error("wait error %s", err)
			log.Info("zc: 1")
			continue
		}
		log.Info("program exited: %s", stat)

		if err := c.logReadPipe.Close(); err != nil {
			log.Error("close log pipe reader: %s", err)
		}
		if err := c.logWritePipe.Close(); err != nil {
			log.Error("close log pipe writer: %s", err)
		}
		if closed {
			return
		}

		if c.getWantStop() {
			// stopped by client, just stay at stopped
		} else {
			// stopped by the program, restart as config specified
			switch processConfig.RestartStrategy {
			case config.RestartStrategyNone:
			case config.RestartStrategyAlways:
				c.handleStart()
			case config.RestartStrategyOnFailure:
				if !stat.Success() {
					c.handleStart()
				}
			}
		}
	}
}

func (c *Controller) handleStart() (err error) {
	if err = c.startAction(); err == nil {
		log.Info("started program")
	} else {
		log.Error("start program: %s", err)
	}
	return
}

func (c *Controller) handleStop() (err error) {
	if err = c.stopAction(); err == nil {
		log.Info("stopped program")
	} else {
		log.Error("stop program: %s", err)
	}
	return
}

func (c *Controller) handleRestart() (err error) {
	defer func() {
		if err == nil {
			log.Info("restarted program")
		} else {
			log.Error("restart program: %s", err)
		}
	}()

	if err = c.stopAction(); err != nil {
		return err
	}
	for {
		if !c.running() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err = c.startAction(); err != nil {
		return err
	}
	return
}

func (c *Controller) handleReload() (err error) {
	defer func() {
		if err == nil {
			log.Info("reloaded program")
		} else {
			log.Error("reload program: %s", err)
		}
	}()

	if err := c.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		return fmt.Errorf("reload send SIGHUP: %s", err)
	}
	return
}

func (c *Controller) handleKill() (err error) {
	defer func() {
		if err == nil {
			log.Info("killed program")
		} else {
			log.Error("kill program: %s", err)
		}
	}()

	if !c.running() {
		return
	}
	c.setWantStop()
	if err := c.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill: %s", err)
	}
	return
}

func (c *Controller) handleStatus(rsp *Response) error {
	if !c.running() {
		rsp.Message = "process not running.\n"
		return nil
	}

	proc, err := procfs.NewProc(c.cmd.Process.Pid)
	if err != nil {
		rsp.Message = fmt.Sprintf("failed to get /pid/%d: %s", c.cmd.Process.Pid, err)
		return nil
	}

	stat, err := proc.Stat()
	if err != nil {
		rsp.Message = fmt.Sprintf("failed to get /pid/%d/stat: %s", c.cmd.Process.Pid, err)
		return nil
	}

	cmdline, err := proc.CmdLine()
	if err != nil {
		rsp.Message = fmt.Sprintf("failed to get /pid/%d/cmdline: %s", c.cmd.Process.Pid, err)
		return nil
	}

	rsp.Message = fmt.Sprintf("%s %d %s\n", stat.State, stat.PID, strings.Join(cmdline, " "))
	return nil
}

func (c *Controller) startAction() error {
	if c.running() {
		return nil
	}
	c.cmd.Process = nil

	c.logReadPipe, c.logWritePipe = io.Pipe()
	c.cmd.Stdout = c.logWritePipe
	c.cmd.Stderr = c.logWritePipe

	go func() {
		written, err := io.Copy(c.logger, c.logReadPipe)
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			log.Error("stopped logger harvest, written %d bytes, err %s", written, err)
		}
	}()

	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("start command: %s", err)
	}
	c.startWait()
	return nil
}

func (c *Controller) stopAction() error {
	if !c.running() {
		return nil
	}
	c.setWantStop()
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %s", err)
	}
	return nil
}

func (c *Controller) setWantStop() {
	atomic.StoreInt32(&c.wantStop, 1)
}

func (c *Controller) getWantStop() bool {
	return atomic.CompareAndSwapInt32(&c.wantStop, 1, 0)
}

func (c *Controller) startWait() {
	go func() { c.waiterCh <- struct{}{} }()
}

func (c *Controller) running() bool {
	if c.cmd.Process == nil {
		return false
	}
	return c.processState() != nil
}

func (c *Controller) processState() *procfs.ProcStat {
	proc, err := procfs.NewProc(c.cmd.Process.Pid)
	if err != nil {
		return nil
	}
	procState, _ := proc.Stat()
	return &procState
}
