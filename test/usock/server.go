package usock

import (
	"encoding/gob"
	"fmt"
	"net"
	"strings"

	"github.com/sequix/sup/pkg/log"
	"github.com/sequix/sup/pkg/util"
)

type Server struct {
	socketPath string
	reqExample interface{}
	ln         *net.UnixListener
	connCh     chan *net.UnixConn
	requestCh  chan *Request
}

func NewServer(socketPath string) (*Server, error) {
	ua, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, err
	}

	ln, err := net.ListenUnix("unix", ua)
	if err != nil {
		return nil, err
	}
	ln.SetUnlinkOnClose(true)

	return &Server{
		socketPath: socketPath,
		ln:         ln,
		connCh:     make(chan *net.UnixConn),
		requestCh:  make(chan *Request),
	}, nil
}

func (s *Server) Run(stop util.BroadcastCh) {
	go s.acceptor()
	for {
		select {
		case uc := <-s.connCh:
			go func() { s.requestCh <- &Request{uc} }()
		case <-stop:
			if err := s.ln.Close(); err != nil {
				log.Error("close %q: %s", s.socketPath, err)
			}
			close(s.connCh)
			close(s.requestCh)
			return
		}
	}
}

func (s *Server) acceptor() {
	for {
		uc, err := s.ln.AcceptUnix()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Error("accept conn from %q: %s", s.socketPath, err)
			continue
		}
		s.connCh <- uc
	}
}

func (s *Server) RequestCh() <-chan *Request {
	return s.requestCh
}

type Request struct {
	conn *net.UnixConn
}

func (r *Request) DecodeInto(req interface{}) error {
	dec := gob.NewDecoder(r.conn)
	if err := dec.Decode(req); err != nil {
		return fmt.Errorf("decode request to %T: %s", req, err)
	}
	return nil
}

func (r *Request) Respond(rsp interface{}) error {
	enc := gob.NewEncoder(r.conn)
	if err := enc.Encode(rsp); err != nil {
		return fmt.Errorf("write response error: %s", err)
	}
	if err := r.conn.Close(); err != nil {
		return fmt.Errorf("close conn: %s", err)
	}
	return nil
}
