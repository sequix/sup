package test

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
	"os/exec"
	"strings"
	"testing"

	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/meta"
)

var (
	server     *rpc.Server
	controller *Controller
)

func TestA(t *testing.T) {
	controller = &Controller{}
	server = rpc.NewServer()
	if err := server.Register(controller); err != nil {
		log.Fatal("registry controller to rpc: %s", err)
	}
	socketPath := "/tmp/test.sock"
	ua, _ := net.ResolveUnixAddr("unix", socketPath)

	ln, err := net.ListenUnix("unix", ua)
	panicIfErr(err)
	ln.SetUnlinkOnClose(true)
	defer ln.Close()

	go func() {
		for {
			uc, err := ln.Accept()
			if err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") {
					return
				}
				log.Error("accept conn: %s", err)
				continue
			}
			go server.ServeConn(uc)
		}
	}()

	c, err := rpc.Dial("unix", socketPath)
	panicIfErr(err)

	rsp := &meta.Response{}
	err = c.Call("Controller.Start", &meta.Request{}, rsp)
	fmt.Printf("%T: %#v", err, err)
	fmt.Println(rsp)
}

type Controller struct {
	cmd *exec.Cmd
}

func (c *Controller) Start(req meta.Request, rsp *meta.Response) error {
	fmt.Println("Start")
	rsp.Status = meta.ResponseStatusOk
	rsp.Message = "message"
	return errors.New("zctest")
}

func (c *Controller) Stop(req meta.Request, rsp *meta.Response) error {
	fmt.Println("Stop")
	return nil
}

func (c *Controller) Restart(req meta.Request, rsp *meta.Response) error {
	fmt.Println("Restart")
	return nil
}

func (c *Controller) Reload(req meta.Request, rsp *meta.Response) error {
	fmt.Println("Reload")
	return nil
}

func (c *Controller) Kill(req meta.Request, rsp *meta.Response) error {
	fmt.Println("Kill")
	return nil
}
