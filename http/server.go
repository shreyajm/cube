package http

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/anuvu/cube/config"
	"github.com/anuvu/cube/service"
)

// Service is the object through which people can register HTTP servers.
type Service interface {
	Register(string, http.Handler)
}

type server struct {
	config  *configuration
	mux     *http.ServeMux
	server  http.Server
	running int32
}

// configuration defines the configurable parameters of http service
type configuration struct {
	// Listen port
	Port int `json:"port"`
}

// New creates a new HTTP Service
func New(ctx service.Context) Service {
	return &server{config: &configuration{}, mux: http.NewServeMux()}
}

func (s *server) Register(url string, h http.Handler) {
	s.mux.Handle(url, h)
}

func (s *server) Configure(ctx service.Context, store config.Store) error {
	if err := store.Get("http", s.config); err != nil {
		return err
	}
	return nil
}

func (s *server) Start(ctx service.Context) error {
	addr := fmt.Sprintf("localhost:%d", s.config.Port)
	s.server = http.Server{Addr: addr, Handler: s.mux}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	atomic.AddInt32(&s.running, 1)
	go func() {
		if err := s.server.Serve(l); err != nil {
			ctx.Log().Info().Error(err).Msg("serve stopping")
		}
	}()
	return nil
}

func (s *server) Stop(ctx service.Context) error {
	atomic.AddInt32(&s.running, -1)
	return s.server.Close()
}

func (s *server) IsHealthy(ctx service.Context) bool {
	return atomic.LoadInt32(&s.running) > 0
}
