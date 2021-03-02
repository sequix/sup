package usock

import (
	"fmt"
	"testing"

	"github.com/sequix/sup/pkg/util"
)

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

type testRequest struct {
	Name string
}

type testResponse struct {
	Msg string
}

func Test(t *testing.T) {
	socketPath := "/tmp/test.sock"
	s, err := NewServer(socketPath)
	panicIfErr(err)
	rw := util.Run(s.Run)
	defer rw.StopAndWait()

	c, err := NewClient(socketPath)
	panicIfErr(err)

	go func() {
		req := <-s.RequestCh()
		reqBody := &testRequest{}
		panicIfErr(req.DecodeInto(reqBody))
		panicIfErr(req.Respond(&testResponse{Msg: fmt.Sprintf("hi %s", reqBody.Name)}))
	}()

	req := &testRequest{"sequix"}
	rsp := &testResponse{}

	panicIfErr(c.Send(req, &rsp))
	if rsp.Msg != "hi sequix" {
		t.Fatalf("not expected rsp, got %#v", rsp)
	}
}
