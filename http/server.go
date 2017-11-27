package http

import (
	"fmt"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/anuvu/cube/component"
	"github.com/anuvu/cube/config"
)

// Server is the object through which people can register HTTP servers.
type Server interface {
	Register(string, http.Handler)
}

type server struct {
	config  *configuration
	mux     *http.ServeMux
	server  http.Server
	running int32
}

// configuration defines the configurable parameters of http server
type configuration struct {
	config.BaseConfig
	// Listen port
	Port int `json:"port"`
}

// New creates a new HTTP server
func New(ctx component.Context) Server {
	cfg := &configuration{
		config.BaseConfig{ConfigKey: "http"},
		0,
	}
	return &server{
		config: cfg,
		mux:    http.NewServeMux(),
	}
}

func (s *server) Register(url string, h http.Handler) {
	s.mux.Handle(url, h)
}

func (s *server) Config() config.Config {
	return s.config
}

func (s *server) Configure(ctx component.Context) error {
	return nil
}

func (s *server) Start(ctx component.Context) error {
	addr := fmt.Sprintf("localhost:%d", s.config.Port)
	s.server = http.Server{Addr: addr, Handler: s.mux}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	atomic.AddInt32(&s.running, 1)
	go func() {
		if err := s.server.Serve(l); err != nil {
			ctx.Log().Info().Error(err).Msg("error starting server")
		}
	}()
	return nil
}

func (s *server) Stop(ctx component.Context) error {
	atomic.AddInt32(&s.running, -1)
	return s.server.Close()
}

func (s *server) IsHealthy(ctx component.Context) bool {
	return atomic.LoadInt32(&s.running) > 0
}
