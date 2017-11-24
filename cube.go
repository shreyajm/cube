package cube

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/anuvu/cube/component"
	"github.com/anuvu/cube/signal"
)

// ServerInit provides the server initialization function type.
// This function is called to customize server initialization.
type ServerInit func(g component.Group) error

// Main is the entrypoint of the server that can be customized by providing a
// ServerInit function. Developers can create custom components and component
// groups in this function.
//
// By default a signal handler is installed to handle SIGINT and SIGTERM for
// graceful shutdown of the server.
func Main(initF ServerInit) {
	name := filepath.Base(os.Args[0])
	base := component.New(name + "-core")
	base.Add(signal.New)

	// Install the signal handler
	srvGrp := base.New(name)
	srvGrp.Add(newShutHandler)

	// Initialize all the server components
	if err := initF(srvGrp); err != nil {
		panic(err)
	}

	// Configure the server
	if err := base.Configure(); err != nil {
		panic(err)
	}

	// Start the server
	if err := base.Start(); err != nil {
		panic(err)
	}

	// Wait for shutdown sequence to be initiated by someone
	base.Invoke(func(ctx component.Context) {
		<-ctx.Ctx().Done()
	})

	// Stop all the components and exit
	if err := base.Stop(); err != nil {
		panic(err)
	}
}

type shutDownHandler struct {
	ctx      component.Context
	router   signal.Router
	shutFunc component.ServerShutdown
}

func newShutHandler(ctx component.Context, router signal.Router, shutFunc component.ServerShutdown) *shutDownHandler {
	s := &shutDownHandler{ctx, router, shutFunc}
	s.router.Handle(syscall.SIGINT, s.shut)
	s.router.Handle(syscall.SIGTERM, s.shut)
	return s
}

func (s *shutDownHandler) shut(sig os.Signal) {
	s.ctx.Log().Info().Str("signal", sig.String()).Msg("Attempting a graceful server shutdown.")
	s.shutFunc()
}
