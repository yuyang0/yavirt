package httpserver

import (
	"context"
	"net"
	"net/http"

	"github.com/projecteru2/yavirt/configs"
	"github.com/projecteru2/yavirt/internal/metrics"
	"github.com/projecteru2/yavirt/internal/service"
	"github.com/projecteru2/yavirt/pkg/errors"
	"github.com/projecteru2/yavirt/pkg/log"
)

// HTTPServer .
type HTTPServer struct {
	Service    service.Service
	httpServer *http.Server
	quit       chan struct{}
}

// Listen .
func New(svc service.Service, quit chan struct{}) (srv *HTTPServer) {
	srv = &HTTPServer{
		Service: svc,
		quit:    quit,
	}
	srv.httpServer = srv.newHTTPServer()
	return
}

func (s *HTTPServer) newHTTPServer() *http.Server {
	var mux = http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.Handle("/", newAPIHandler(s.Service))
	return &http.Server{Handler: mux} //nolint
}

// Reload .
func (s *HTTPServer) Reload() error {
	return nil
}

// Serve .
func (s *HTTPServer) Serve() (err error) {
	defer func() {
		log.Warnf("[httpserver] main loop %p exit", s)
	}()
	lis, err := net.Listen("tcp", configs.Conf.BindHTTPAddr)
	if err != nil {
		return
	}
	var errCh = make(chan error, 1)
	go func() {
		defer func() {
			log.Warnf("[httpserver] HTTP server %p exit", s.httpServer)
		}()
		errCh <- s.httpServer.Serve(lis)
	}()

	select {
	case <-s.quit:
		return nil
	case err = <-errCh:
		return errors.Trace(err)
	}
}

// Stop .
func (s *HTTPServer) Stop(force bool) {
	var err error
	defer func() {
		if err != nil {
			log.ErrorStack(err)
			metrics.IncrError()
		}
	}()

	var ctx, cancel = context.WithTimeout(context.Background(), configs.Conf.GracefulTimeout.Duration())
	defer cancel()

	if err = s.httpServer.Shutdown(ctx); err != nil {
		return
	}
}
