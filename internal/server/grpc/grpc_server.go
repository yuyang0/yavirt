package grpcserver

import (
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/projecteru2/libyavirt/grpc/gen"
	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/pkg/log"
)

// GRPCServer .
type GRPCServer struct {
	server *grpc.Server
	app    pb.YavirtdRPCServer
	quit   chan struct{}
}

func New(svc service.Service, quit chan struct{}) *GRPCServer {
	srv := &GRPCServer{
		server: grpc.NewServer(),
		app:    &GRPCYavirtd{service: svc},
		quit:   quit,
	}
	reflection.Register(srv.server)

	return srv
}

// Reload .
func (s *GRPCServer) Reload() error {
	return nil
}

// Serve .
func (s *GRPCServer) Serve() error {
	defer func() {
		log.Warnf("[grpcserver] main loop %p exit", s)
	}()
	lis, err := net.Listen("tcp", configs.Conf.BindGRPCAddr)
	if err != nil {
		return err
	}
	pb.RegisterYavirtdRPCServer(s.server, s.app)

	return s.server.Serve(lis)
}

// Close .
func (s *GRPCServer) Stop(force bool) {
	if force {
		log.Warnf("[grpcserver] terminate grpc server forcefully")
		s.server.Stop()
		return
	}

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
}
