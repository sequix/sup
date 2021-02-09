package process

import (
	"fmt"
	"net/rpc"

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

func Stop() error {
	return client.Call("Controller.Stop", &Request{}, &Response{})
}

func Restart() error {
	return client.Call("Controller.Restart", &Request{}, &Response{})
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
