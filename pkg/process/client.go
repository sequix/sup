package process

import (
	"fmt"
	"net/rpc"
	"os"
	"syscall"
	"time"

	"github.com/sequix/sup/pkg/config"
	"github.com/sequix/sup/pkg/log"
)

var (
	client *rpc.Client
)

func InitClient() {
	var (
		err        error
		socketPath = config.G.SupConfig.Socket
	)
	client, err = rpc.Dial("unix", socketPath)
	if err != nil {
		log.Fatal("dial %s: %s", socketPath, err)
	}
}

func ClientClose() {
	if err := client.Close(); err != nil {
		log.Error("close client: %s", err)
	}
}

func Start() error {
	return client.Call("Controller.Start", &Request{}, &Response{})
}

func StartWait() error {
	return client.Call("Controller.StartWait", &Request{}, &Response{})
}

func Stop() error {
	return client.Call("Controller.Stop", &Request{}, &Response{})
}

func StopWait() error {
	return client.Call("Controller.StopWait", &Request{}, &Response{})
}

func Restart() error {
	return client.Call("Controller.Restart", &Request{}, &Response{})
}

func RestartWait() error {
	return client.Call("Controller.RestartWait", &Request{}, &Response{})
}

func Reload() error {
	return client.Call("Controller.Reload", &Request{}, &Response{})
}

func Kill() error {
	return client.Call("Controller.Kill", &Request{}, &Response{})
}

func Status() error {
	rsp := &Response{}
	if err := client.Call("Controller.Status", &Request{}, &rsp); err != nil {
		return err
	}
	fmt.Print(rsp.Message)
	return nil
}

func Exit() error {
	rsp := &Response{}
	if err := client.Call("Controller.SupPid", &Request{}, &rsp); err != nil {
		return err
	}
	sup, err := os.FindProcess(rsp.SupPid)
	if err != nil {
		return fmt.Errorf("finding sup process: %s", err)
	}
	if err := sup.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("sending SIGTERM to sup process %d: %s", rsp.SupPid, err)
	}
	return nil
}

func ExitWait() error {
	if err := Exit(); err != nil {
		return err
	}
	socketPath := config.G.SupConfig.Socket
	for {
		_, err := os.Stat(socketPath)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("stat socket %s: %s", socketPath, err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
