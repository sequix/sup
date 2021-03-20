package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/rotate"
	"github.com/sequix/sup/pkg/util"
)

type Controller struct {
	cmd          *exec.Cmd
	logWritePipe *io.PipeWriter
	logReadPipe  *io.PipeReader
	logger       *rotate.FileWriter
	actionMu     sync.Mutex
	waiterCh     chan struct{}
	wantStop     int32
	wantExit     int32
}

func (c *Controller) close() {
	log.Info("stopping the program")
	if err := controller.stopAction(); err != nil {
		log.Error("stop the program: %s", err)
	}
	if err := c.logger.Close(); err != nil {
		log.Error("stop the rotate logger: %s", err)
	}
}

func (c *Controller) Start(_ *Request, _ *Response) error {
	log.Info("recv start action")
	go func() {
		if err := c.handleStart(); err != nil {
			log.Error(err.Error())
		}
	}()
	return nil
}

func (c *Controller) StartWait(_ *Request, _ *Response) error {
	log.Info("recv start-wait action")
	if err := c.handleStart(); err != nil {
		log.Error(err.Error())
	}
	return nil
}

func (c *Controller) Stop(_ *Request, _ *Response) error {
	log.Info("recv stop action")
	go func() {
		if err := c.handleStop(); err != nil {
			log.Error(err.Error())
		}
	}()
	return nil
}

func (c *Controller) StopWait(_ *Request, _ *Response) error {
	log.Info("recv stop-wait action")
	if err := c.handleStop(); err != nil {
		log.Error(err.Error())
	}
	return nil
}

func (c *Controller) Restart(_ *Request, _ *Response) error {
	log.Info("recv restart action")
	go func() {
		if err := c.handleRestart(); err != nil {
			log.Error(err.Error())
		}
	}()
	return nil
}

func (c *Controller) RestartWait(_ *Request, _ *Response) error {
	log.Info("recv restart-wait action")
	if err := c.handleRestart(); err != nil {
		log.Error(err.Error())
	}
	return nil
}

func (c *Controller) Reload(_ *Request, _ *Response) error {
	log.Info("recv reload action")
	if err := c.handleReload(); err != nil {
		log.Error(err.Error())
	}
	return nil
}

func (c *Controller) Kill(_ *Request, _ *Response) error {
	log.Info("recv kill action")
	if err := c.handleKill(); err != nil {
		log.Error(err.Error())
	}
	return nil
}

func (c *Controller) Status(_ *Request, rsp *Response) error {
	log.Info("recv status action")
	err := c.handleStatus(rsp)
	if err != nil {
		log.Error(err.Error())
	}
	return err
}

func (c *Controller) SupPid(_ *Request, rsp *Response) error {
	rsp.SupPid = os.Getpid()
	return nil
}

func (c *Controller) waiter(stop util.BroadcastCh) {
	go func() {
		<-stop
		c.setWantExit()
		close(c.waiterCh)
	}()

	for {
		<-c.waiterCh
		if c.getWantExit() {
			return
		}
		if c.getWantStop() {
			continue
		}

		stat, err := c.cmd.Process.Wait()
		if err != nil {
			log.Error("wait error %s", err)
			continue
		}
		log.Info("program exited: %s", stat)

		if err := c.logReadPipe.Close(); err != nil {
			log.Error("close log pipe reader: %s", err)
		}
		if err := c.logWritePipe.Close(); err != nil {
			log.Error("close log pipe writer: %s", err)
		}

		// Wait for wantExit and wantStop set.
		time.Sleep(300 * time.Millisecond)
		if c.getWantExit() {
			return
		}
		if c.getWantStop() {
			continue
		}

		// program stopped itself, restart as config specified
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
	c.waitNotRunning()
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
		rsp.Message = "NotStarted\n"
		return nil
	}
	// procfs doc: https://man7.org/linux/man-pages/man5/procfs.5.html
	pid := c.cmd.Process.Pid
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statBytes, err := ioutil.ReadFile(statPath)
	if err != nil {
		rsp.Message = fmt.Sprintf("failed to read %s: %s", statPath, err)
		return nil
	}
	statFields := bytes.Split(statBytes, []byte(" "))
	if len(statFields) < 3 {
		rsp.Message = fmt.Sprintf("want at least 3 proc stat field, got %d", len(statFields))
		return nil
	}
	cmdlinePath := fmt.Sprintf("/proc/%d/cmdline", pid)
	cmdline, err := ioutil.ReadFile(cmdlinePath)
	if err != nil {
		rsp.Message = fmt.Sprintf("failed to read %s: %s", cmdlinePath, err)
		return nil
	}
	cmdline = bytes.ReplaceAll(cmdline, []byte{0}, []byte(" "))
	rsp.Message = fmt.Sprintf("%s %d %s\n", string(statFields[2]), pid, string(cmdline))
	return nil
}

func (c *Controller) startAction() error {
	c.actionMu.Lock()
	defer c.actionMu.Unlock()
	if c.running() {
		return nil
	}
	log.Info("starting program")
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
	time.Sleep(time.Duration(config.G.ProgramConfig.Process.StartSeconds) * time.Second)
	c.startWait()
	return nil
}

func (c *Controller) stopAction() error {
	c.actionMu.Lock()
	defer c.actionMu.Unlock()
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

func (c *Controller) setWantExit() {
	atomic.StoreInt32(&c.wantExit, 1)
}

func (c *Controller) getWantExit() bool {
	return atomic.CompareAndSwapInt32(&c.wantExit, 1, 0)
}

func (c *Controller) startWait() {
	go func() { c.waiterCh <- struct{}{} }()
}

func (c *Controller) running() bool {
	if c.cmd.Process == nil {
		return false
	}
	statPath := fmt.Sprintf("/proc/%d/stat", c.cmd.Process.Pid)
	_, err := os.Open(statPath)
	return err == nil
}

func (c *Controller) waitNotRunning() {
	for {
		if !c.running() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
