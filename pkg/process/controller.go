package process

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/rotate"
)

type Controller struct {
	mu           sync.Mutex
	cmd          *exec.Cmd
	logWritePipe *io.PipeWriter
	logReadPipe  *io.PipeReader
	logger       *rotate.FileWriter
	startedCh    chan struct{}
	exitedCh     chan struct{}
	wantStop     int32
	wantExit     int32
}

func (c *Controller) run(stop <-chan struct{}) {
	for {
		select {
		case <-stop:
			c.setWantExit()
			_ = c.Stop(nil, nil)
			return
		case <-c.startedCh:
			go c.wait()
		case <-c.exitedCh:
			if c.getWantExit() {
				return
			}
			if c.getWantStop() {
				continue
			}
			switch processConfig.RestartStrategy {
			case config.RestartStrategyNone:
			case config.RestartStrategyAlways:
				c.mustStart()
			case config.RestartStrategyOnFailure:
				if !c.cmd.ProcessState.Success() {
					c.mustStart()
				}
			}
		}
	}
}

func (c *Controller) mustStart() {
	for {
		if err := c.startHandler(); err == nil {
			return
		}
	}
}

func (c *Controller) Start(_ *Request, _ *Response) (err error) {
	c.setWantStop(0)
	return c.startHandler()
}

func (c *Controller) startHandler() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Info("starting program")
	if err = c.startAction(); err == nil {
		log.Info("started program %d", c.cmd.Process.Pid)
	} else {
		log.Error("start program: %s", err)
	}
	return
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
		return fmt.Errorf("start program: %s", err)
	}
	time.Sleep(time.Duration(config.G.ProgramConfig.Process.StartSeconds) * time.Second)
	if !c.running() {
		_ = c.logReadPipe.Close()
		_ = c.logWritePipe.Close()
		return fmt.Errorf("program not running after %d seconds", config.G.ProgramConfig.Process.StartSeconds)
	}
	go func() { c.startedCh <- struct{}{} }()
	return nil
}

func (c *Controller) wait() {
	var (
		err  error
		stat *os.ProcessState
	)
	for {
		stat, err = c.cmd.Process.Wait()
		if err != nil {
			log.Warn("wait program %d: %s", c.cmd.Process.Pid, err)
			if c.running() {
				continue
			} else {
				break
			}
		}
	}
	log.Info("program %d exited with stat: %s", c.cmd.Process.Pid, stat)
	if err := c.logReadPipe.Close(); err != nil {
		log.Error("close log pipe reader: %s", err)
	}
	if err := c.logWritePipe.Close(); err != nil {
		log.Error("close log pipe writer: %s", err)
	}
	go func() { c.exitedCh <- struct{}{} }()
}

func (c *Controller) Stop(_ *Request, _ *Response) error {
	c.setWantStop(1)
	return c.stopHandler()
}

func (c *Controller) stopHandler() (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Info("stopping program %d", c.cmd.Process.Pid)
	if err = c.stopAction(); err == nil {
		log.Info("stopped program %d", c.cmd.Process.Pid)
	} else {
		log.Error("stop program %d: %s", c.cmd.Process.Pid, err)
	}
	return
}

func (c *Controller) stopAction() error {
	if !c.running() {
		return nil
	}
	children, err := c.listChildrenProcesses(c.cmd.Process.Pid)
	if err != nil {
		log.Error("failed to list children processes of program %d: %s", c.cmd.Process.Pid, err)
	}
	for _, pid := range children {
		if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to send SIGTERM to grandchild process %d: %s", pid, err)
		}
		log.Info("sent SIGTERM to child process %d", pid)
	}
	if err := c.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %s", err)
	}
	log.Info("sent SIGTERM to child process %d", c.cmd.Process.Pid)
	c.waitNotRunning()
	return nil
}

func (c *Controller) Restart(_ *Request, _ *Response) (err error) {
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		if err == nil {
			log.Info("restarted program %d", c.cmd.Process.Pid)
		} else {
			log.Error("restart program %d: %s", c.cmd.Process.Pid, err)
		}
	}()
	c.setWantStop(0)
	log.Info("restarting program %d", c.cmd.Process.Pid)
	if err = c.stopAction(); err != nil {
		return
	}
	if err = c.startAction(); err != nil {
		return
	}
	return
}

func (c *Controller) Reload(_ *Request, _ *Response) (err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	log.Info("reloading program %d", c.cmd.Process.Pid)
	if err = c.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		log.Error("reload program %d: %s", c.cmd.Process.Pid, err)
	} else {
		log.Info("reloaded program %d", c.cmd.Process.Pid)
	}
	return
}

func (c *Controller) Kill(_ *Request, _ *Response) (err error) {
	c.mu.Lock()
	defer func() {
		c.mu.Unlock()
		if err == nil {
			log.Info("killed program %d", c.cmd.Process.Pid)
		} else {
			log.Error("kill program %d: %s", c.cmd.Process.Pid, err)
		}
	}()
	c.setWantStop(1)
	log.Info("killing program %d", c.cmd.Process.Pid)
	if c.running() {
		children, lerr := c.listChildrenProcesses(c.cmd.Process.Pid)
		if lerr != nil {
			log.Error("failed to list children processes of program %d: %s", c.cmd.Process.Pid, lerr)
		}
		err = c.cmd.Process.Kill()
		if err != nil {
			err = fmt.Errorf("failed to kill child process %d: %s", &c.cmd.Process.Pid, err)
			return
		}
		for _, pid := range children {
			if err = syscall.Kill(pid, syscall.SIGKILL); err != nil {
				err = fmt.Errorf("failed to kill grand-child process %d: %s", pid, err)
				return
			}
			log.Info("killed child process %d", pid)
		}
	}
	c.waitNotRunning()
	return
}

func (c *Controller) Status(_ *Request, rsp *Response) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.running() {
		rsp.Message = "NotStarted\n"
		return nil
	}
	// procfs doc: https://man7.org/linux/man-pages/man5/procfs.5.html
	pid := c.cmd.Process.Pid
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	statBytes, err := os.ReadFile(statPath)
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
	cmdline, err := os.ReadFile(cmdlinePath)
	if err != nil {
		rsp.Message = fmt.Sprintf("failed to read %s: %s", cmdlinePath, err)
		return nil
	}
	cmdline = bytes.ReplaceAll(cmdline, []byte{0}, []byte(" "))
	rsp.Message = fmt.Sprintf("%s %d %s\n", string(statFields[2]), pid, string(cmdline))
	return nil
}

func (c *Controller) SupPid(_ *Request, rsp *Response) error {
	rsp.SupPid = os.Getpid()
	return nil
}

func (c *Controller) running() bool {
	if c.cmd.Process == nil {
		return false
	}
	pids, _ := c.listChildrenProcesses(c.cmd.Process.Pid)
	for _, pid := range append(pids, c.cmd.Process.Pid) {
		if isPidRunning(pid) {
			return true
		}
	}
	return false
}

func (c *Controller) waitNotRunning() {
	for {
		if !c.running() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (c *Controller) setWantStop(ws int32) {
	atomic.StoreInt32(&c.wantStop, ws)
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

var reAllDigits = regexp.MustCompile(`^[0-9]+$`)

func (c *Controller) listChildrenProcesses(ppid int) ([]int, error) {
	var children []int
	fis, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to read dir /proc: %s", err)
	}
	for _, fi := range fis {
		if fi.IsDir() && reAllDigits.MatchString(filepath.Base(fi.Name())) {
			pidS := filepath.Base(fi.Name())
			statusBytes, err := os.ReadFile(filepath.Join("/proc", pidS, "status"))
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(statusBytes), "\n") {
				line = strings.ToLower(strings.TrimSpace(line))
				if strings.HasPrefix(line, "ppid:") {
					fields := strings.Fields(line)
					gotPpid, err := strconv.Atoi(fields[len(fields)-1])
					if err != nil {
						continue
					}
					if gotPpid == ppid {
						pid, _ := strconv.Atoi(pidS)
						children = append(children, pid)
					}
					break
				}
			}
		}
	}
	return children, nil
}
