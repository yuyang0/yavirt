package server

import (
	"net"
	"sync"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/pkg/netx"
)

// Server .
type Serverable interface {
	Reload() error
	Serve() error
	Close()
	ExitCh() chan struct{}
}

// Server .
type Server struct {
	Addr     string
	Listener net.Listener
	Exit     struct {
		sync.Once
		Ch chan struct{}
	}
}

// Listen .
func Listen(addr string) (srv *Server, err error) {
	srv = &Server{}
	srv.Exit.Ch = make(chan struct{}, 1)
	srv.Listener, srv.Addr, err = srv.Listen(addr)
	return
}

// Listen .
func (s *Server) Listen(addr string) (lis net.Listener, ip string, err error) {
	var network = "tcp"
	if lis, err = net.Listen(network, addr); err != nil {
		return
	}

	if ip, err = netx.GetOutboundIP(configs.Conf.Core.Addrs[0]); err != nil {
		return
	}

	return
}
