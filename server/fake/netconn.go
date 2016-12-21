package fake

import (
	"fmt"
	"io"
	"net"
	"time"
)

type Addr struct {
	network string
	host    string
	port    int
}

func (a *Addr) Network() string {
	return a.network
}
func (a *Addr) String() string {
	return fmt.Sprintf("%s://%s:%d", a.network, a.host, a.port)
}

type NetConn struct {
	W       func([]byte) (int, error)
	R       io.Reader
	Network string
	Host    string
	Port    int
}

func (f *NetConn) Read(b []byte) (int, error) {
	// 給hub 測試用
	return f.R.Read(b)
}
func (f *NetConn) Write(b []byte) (int, error) {
	return f.W(b)
}
func (*NetConn) Close() error                     { return nil }
func (*NetConn) LocalAddr() net.Addr              { return nil }
func (f *NetConn) RemoteAddr() net.Addr           { return &Addr{network: f.Network, host: f.Host, port: f.Port} }
func (*NetConn) SetDeadline(time.Time) error      { return nil }
func (*NetConn) SetReadDeadline(time.Time) error  { return nil }
func (*NetConn) SetWriteDeadline(time.Time) error { return nil }
