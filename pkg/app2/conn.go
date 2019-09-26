package app2

import (
	"net"
	"time"

	"github.com/skycoin/skywire/pkg/app2/network"
)

// Conn is a connection from app client to the server.
// Implements `net.Conn`.
type Conn struct {
	id       uint16
	rpc      RPCClient
	local    network.Addr
	remote   network.Addr
	freeConn func()
}

func (c *Conn) Read(b []byte) (int, error) {
	n, err := c.rpc.Read(c.id, b)
	if err != nil {
		return 0, err
	}

	return n, err
}

func (c *Conn) Write(b []byte) (int, error) {
	return c.rpc.Write(c.id, b)
}

func (c *Conn) Close() error {
	defer func() {
		if c.freeConn != nil {
			c.freeConn()
		}
	}()

	return c.rpc.CloseConn(c.id)
}

func (c *Conn) LocalAddr() net.Addr {
	return c.local
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.remote
}

func (c *Conn) SetDeadline(t time.Time) error {
	return errMethodNotImplemented
}

func (c *Conn) SetReadDeadline(t time.Time) error {
	return errMethodNotImplemented
}

func (c *Conn) SetWriteDeadline(t time.Time) error {
	return errMethodNotImplemented
}
