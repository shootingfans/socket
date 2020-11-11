package socket

import (
	"net"
	"sync/atomic"
	"time"
)

type Connection struct {
	net.Conn
	rBytes uint64
	wBytes uint64
	cnesc  int64
	enesc  int64
}

func (c *Connection) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		atomic.AddUint64(&c.rBytes, uint64(n))
	}
	return
}

func (c *Connection) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		atomic.AddUint64(&c.wBytes, uint64(n))
	}
	return
}

func (c *Connection) Close() error {
	if atomic.CompareAndSwapInt64(&c.enesc, 0, time.Now().UnixNano()) {
		return c.Conn.Close()
	}
	return nil
}

func (c *Connection) Created() time.Time {
	return time.Unix(0, c.cnesc)
}

func (c *Connection) Duration() time.Duration {
	return time.Unix(0, c.enesc).Sub(time.Unix(0, c.cnesc))
}

func (c *Connection) ReadBytes() uint64 {
	return atomic.LoadUint64(&c.rBytes)
}

func (c *Connection) WriteBytes() uint64 {
	return atomic.LoadUint64(&c.wBytes)
}

func NewConnection(conn net.Conn) *Connection {
	return &Connection{
		Conn:   conn,
		rBytes: 0,
		wBytes: 0,
		cnesc:  time.Now().UnixNano(),
		enesc:  0,
	}
}
