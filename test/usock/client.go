package usock

import (
	"encoding/gob"
	"fmt"
	"net"

	"github.com/sequix/sup/pkg/log"
)

type Client struct {
	socketPath string
	conn       *net.UnixConn
}

func NewClient(socketPath string) (*Client, error) {
	ua, err := net.ResolveUnixAddr("unix", socketPath)
	if err != nil {
		return nil, err
	}

	uc, err := net.DialUnix("unix", nil, ua)
	if err != nil {
		return nil, err
	}

	return &Client{
		socketPath: socketPath,
		conn:       uc,
	}, nil
}

func (c *Client) Close() {
	if err := c.conn.Close(); err != nil {
		log.Error("close conn to %q: %s", c.socketPath, err)
	}
}

func (c *Client) Send(req, rsp interface{}) error {
	enc := gob.NewEncoder(c.conn)
	if err := enc.Encode(req); err != nil {
		return fmt.Errorf("write to %s: %s", c.socketPath, err)
	}
	dec := gob.NewDecoder(c.conn)
	if err := dec.Decode(rsp); err != nil {
		return fmt.Errorf("read from %s: %s", c.socketPath, err)
	}
	return nil
}
