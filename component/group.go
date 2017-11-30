package component

import (
	"context"
	"flag"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/anuvu/cube/config"
	"github.com/anuvu/cube/di"
	"github.com/anuvu/zlog"
)

// ConfigHook is the interface that provides the configuration callback for the component.
type ConfigHook interface {
	// Config returns pointer to the object that captures the configuration of the
	// component. The framework will populate this object after retrieving the
	// configuration of the component from the configuration store.
	Config() config.Config

	// Configure callback is issued after the configuration object is retrieved
	// from the store.
	Configure(ctx Context) error
}

// StartHook is the interface that provides the start callback for the component.
type StartHook interface {
	Start(ctx Context) error
}

// StopHook is the interface that provides the stop callback for the component.
type StopHook interface {
	Stop(ctx Context) error
}

// HealthHook is the interface that provides the health callback for the component.
type HealthHook interface {
	IsHealthy(ctx Context) bool
}

// ServerShutdown invokes the server shutdown sequence.
type ServerShutdown context.CancelFunc

// Group provides and interface to add custom components and
// sub-groups to this group.
type Group interface {
	Add(ctr interface{}) error
	Invoke(f interface{}) error
	New(name string) Group
	Create() error
	Configure() error
	Start() error
	Stop() error
	IsHealthy() bool
}

// Group is a group of components, that have inter-dependencies.
type group struct {
	name        string
	parent      *group
	children    map[string]*group
	store       config.Store
	cli         *flag.FlagSet
	c           *di.Container
	ctx         *srvCtx
	configHooks []ConfigHook
	startHooks  []StartHook
	stopHooks   []StopHook
	healthHooks []HealthHook
}

var ctxType = reflect.TypeOf((*Context)(nil)).Elem()
var shutType = reflect.TypeOf((*Shutdown)(nil)).Elem()

// New creates a new component group with the specified parent. If the parent is nil
// this group is the root group.
func New(name string) Group {
	grp := newGroup(name, nil)

	// Root container should provide the server shutdown function
	shut := ServerShutdown(grp.ctx.cancelFunc)
	grp.c.Add(func() ServerShutdown { return shut })

	// Root container should provide cli
	grp.cli = flag.NewFlagSet(name, flag.ContinueOnError)
	grp.c.Add(func() *flag.FlagSet { return grp.cli })

	// Create the store
	grp.store = newConfigStore(grp.cli)
	return grp
}

func newGroup(name string, parent *group) *group {
	var pc *di.Container
	var pctx *srvCtx
	var cli *flag.FlagSet
	var store config.Store
	if parent != nil {
		pc = parent.c
		pctx = parent.ctx
		cli = parent.cli
		store = parent.store
	}

	log := zlog.New(name)
	c := di.New(pc, ctxType, shutType)
	ctx := newContext(pctx, log)
	grp := &group{
		name:        name,
		parent:      parent,
		children:    map[string]*group{},
		store:       store,
		cli:         cli,
		c:           c,
		ctx:         ctx,
		configHooks: []ConfigHook{},
		startHooks:  []StartHook{},
		stopHooks:   []StopHook{},
		healthHooks: []HealthHook{},
	}

	// Provide the Context, Shutdown per group
	grp.c.Add(func() Context { return grp.ctx })
	grp.c.Add(func() Shutdown { return grp.ctx.Shutdown })

	return grp
}

func (g *group) New(name string) Group {
	grp := newGroup(name, g)

	// FIXME: Potential child name collision, check for it.
	g.children[name] = grp

	return grp
}

// Add adds a new component constructor to the component group.
func (g *group) Add(ctr interface{}) error {
	// add the component constructor to the container
	return g.c.Add(ctr)
}

// Invoke invokes a function with dependency injection.
func (g *group) Invoke(f interface{}) error {
	return g.c.Invoke(f, nil)
}

func (g *group) Create() error {
	g.ctx.Log().Info().Msg("creating group")
	// g.c.Create will call this function for each value produced by ctr
	// constructor method we then check if the produced value implements
	// any of the lifecycle hooks and cache them so that we can invoke them
	// as part of the server lifecycle.
	vf := func(v reflect.Value) error {
		return g.addLCHooks(v)
	}
	if err := g.c.Create(vf); err != nil {
		return err
	}

	for _, child := range g.children {
		if err := child.Create(); err != nil {
			return err
		}
	}
	return nil
}

// Configure calls the configure hooks on all components registered for configuration.
func (g *group) Configure() error {
	if g.parent == nil {
		// root group parse the cli and initialize the config store
		if err := g.cli.Parse(os.Args[1:]); err != nil {
			return err
		}
		if err := g.store.Open(); err != nil {
			return err
		}
		defer g.store.Close()
	}

	g.ctx.Log().Info().Msg("configuring group")
	for _, h := range g.configHooks {
		cfg := h.Config()
		if err := g.store.Get(cfg); err != nil {
			return err
		}
		if err := h.Configure(g.ctx); err != nil {
			return err
		}
	}

	// Configure all the child groups.
	for _, child := range g.children {
		if err := child.Configure(); err != nil {
			return err
		}
	}
	return nil
}

// Start calls the start hooks on all components registered for startup.
// If an error occurs on any hook, subsequent start calls are abandoned
// and a best effort stop is initiated.
func (g *group) Start() error {
	g.ctx.Log().Info().Msg("starting group")
	for _, h := range g.startHooks {
		if err := g.c.Invoke(h.Start, nil); err != nil {
			// We need to call all stop hooks and ignore errors
			// as we dont know which components are actually participating
			// in the stop callbacks
			defer g.Stop()
			return err
		}
	}

	// Start all the child groups
	for _, child := range g.children {
		if err := child.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Stop calls the stop hooks on all components registered for shutdown.
func (g *group) Stop() error {
	var e error

	// Stop all the child groups first
	for _, child := range g.children {
		e = child.Stop()
	}

	g.ctx.Log().Info().Msg("stopping group")

	// Invoke the stop hooks in the reverse dependency order
	if len(g.stopHooks) > 0 {
		for i := len(g.stopHooks) - 1; i == 0; i-- {
			h := g.stopHooks[i]
			if err := g.c.Invoke(h.Stop, nil); err != nil {
				// FIXME: We need to make this multi-error
				e = err
			}
		}
	}
	return e
}

// IsHealthy returns true if all components health hooks return true else false
func (g *group) IsHealthy() bool {
	for _, h := range g.healthHooks {
		if !h.IsHealthy(g.ctx) {
			return false
		}
	}

	for _, child := range g.children {
		if !child.IsHealthy() {
			return false
		}
	}
	return true
}

// Add the lifecycle hooks to the group.
func (g *group) addLCHooks(v reflect.Value) error {
	val := v.Interface()
	if i, ok := val.(ConfigHook); ok {
		g.configHooks = append(g.configHooks, i)
	}
	if i, ok := val.(StartHook); ok {
		g.startHooks = append(g.startHooks, i)
	}
	if i, ok := val.(StopHook); ok {
		g.stopHooks = append(g.stopHooks, i)
	}
	if i, ok := val.(HealthHook); ok {
		g.healthHooks = append(g.healthHooks, i)
	}
	return nil
}

func newConfigStore(cli *flag.FlagSet) config.Store {
	s := &cfgStore{}
	cli.StringVar(&s.fileCfg, "config.file", "", "file configuration store")
	cli.StringVar(&s.memCfg, "config.mem", "", "in-memory configuration store")
	return s
}

type cfgStore struct {
	fileCfg string
	memCfg  string
	store   config.Store
}

func (s *cfgStore) Open() error {
	if s.fileCfg != "" {
		r, err := os.Open(s.fileCfg)
		if err != nil {
			return err
		}
		s.store = config.NewJSONStore(r)
		return s.store.Open()
	} else if s.memCfg != "" {
		r := strings.NewReader(s.memCfg)
		s.store = config.NewJSONStore(r)
		return s.store.Open()
	}
	// No config store
	return nil
}

func (s *cfgStore) Close() {
	if s.store != nil {
		s.store.Close()
	}
}

func (s *cfgStore) Get(config config.Config) error {
	if config == nil || config.Key().IsNil() {
		return nil
	}
	if s.store == nil {
		return fmt.Errorf("%s key not found", config.Key())
	}
	return s.store.Get(config)
}
