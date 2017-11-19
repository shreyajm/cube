package cube

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/anuvu/cube/service"
	"github.com/anuvu/cube/signal"
)

// ComponentGroup provides and interface to add custom components and
// sub-groups to the cube server.
type ComponentGroup interface {
	AddComponent(ctr interface{})
	NewSubGroup(name string) ComponentGroup
}

type componentGroup struct {
	s *service.Group
}

func (cg *componentGroup) AddComponent(ctr interface{}) {
	if err := cg.s.AddService(ctr); err != nil {
		panic(err)
	}
}

func (cg *componentGroup) NewSubGroup(name string) ComponentGroup {
	return &componentGroup{service.NewGroup(name, cg.s)}
}

// ServerInit provides the server initialization function type.
// This function is called to customize server initialization.
type ServerInit func(g ComponentGroup)

// Main is the entrypoint of the server that can be customized by providing a
// ServerInit function. Developers can create custom components and component
// groups in this function.
//
// By default a signal handler is installed to handle SIGINT and SIGTERM for
// graceful shutdown of the server.
func Main(initF ServerInit) {
	base := service.NewGroup("cube", nil)
	bg := &componentGroup{base}
	bg.AddComponent(signal.NewSignalRouter)

	// Install the signal handler
	name := filepath.Base(os.Args[0])
	srvGrp := bg.NewSubGroup(name)
	srvGrp.AddComponent(newShutHandler)

	// Initialize all the server components
	initF(srvGrp)

	// Configure the server
	if err := base.Configure(); err != nil {
		panic(err)
	}

	// Start the server
	if err := base.Start(); err != nil {
		panic(err)
	}

	// Wait for shutdown sequence to be initiated by someone
	base.Invoke(func(ctx service.Context) {
		<-ctx.Ctx().Done()
	})

	// Stop all the services and exit
	if err := base.Stop(); err != nil {
		panic(err)
	}
}

type shutDownHandler struct {
	ctx      service.Context
	router   signal.Router
	shutFunc service.ServerShutdown
}

func newShutHandler(ctx service.Context, router signal.Router, shutFunc service.ServerShutdown) *shutDownHandler {
	s := &shutDownHandler{ctx, router, shutFunc}
	s.router.Handle(syscall.SIGINT, s.shut)
	s.router.Handle(syscall.SIGTERM, s.shut)
	return s
}

func (s *shutDownHandler) shut(sig os.Signal) {
	s.ctx.Log().Info().Str("signal", sig.String()).Msg("Attempting a graceful server shutdown.")
	s.shutFunc()
}
