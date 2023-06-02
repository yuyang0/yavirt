package grpcserver

import (
	"time"

	"google.golang.org/grpc"

	pb "github.com/projecteru2/libyavirt/grpc/gen"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/server"
	"github.com/projecteru2/yavirt/pkg/log"
)

// GRPCServer .
type GRPCServer struct {
	*server.Server

	server *grpc.Server
	app    pb.YavirtdRPCServer
}

// Listen .
func Listen(svc *server.Service) (srv *GRPCServer, err error) {
	srv = &GRPCServer{}
	if srv.Server, err = server.Listen(configs.Conf.BindGRPCAddr); err != nil {
		return
	}

	srv.server = grpc.NewServer()
	srv.app = &GRPCYavirtd{service: svc}

	return
}

// Reload .
func (s *GRPCServer) Reload() error {
	return nil
}

// Serve .
func (s *GRPCServer) Serve() error {
	defer func() {
		log.Warnf("[grpcserver] main loop %p exit", s)
		s.Close()
	}()

	pb.RegisterYavirtdRPCServer(s.server, s.app)

	return s.server.Serve(s.Listener)
}

// Close .
func (s *GRPCServer) Close() {
	s.Exit.Do(func() {
		close(s.Exit.Ch)

		gracefulDone := make(chan struct{})
		go func() {
			defer close(gracefulDone)
			s.server.GracefulStop()
		}()

		gracefulTimer := time.NewTimer(configs.Conf.GracefulTimeout.Duration())
		select {
		case <-gracefulDone:
			log.Infof("[grpcserver] terminate grpc server gracefully")
		case <-gracefulTimer.C:
			log.Warnf("[grpcserver] terminate grpc server forcefully")
			s.server.Stop()
		}
	})
}

// ExitCh .
func (s *GRPCServer) ExitCh() chan struct{} {
	return s.Exit.Ch
}
