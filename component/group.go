package component

import (
	"context"
	"reflect"

	"github.com/anuvu/cube/config"
	"github.com/anuvu/cube/di"
	"github.com/anuvu/zlog"
)

// ConfigHook is the interface that provides the configuration call back for the component.
type ConfigHook interface {
	Configure(ctx Context, store config.Store) error
}

// StartHook is the interface that provides the start call back for the component.
type StartHook interface {
	Start(ctx Context) error
}

// StopHook is the interface that provides the stop call back for the component.
type StopHook interface {
	Stop(ctx Context) error
}

// HealthHook is the interface that provides the health call back for the component.
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
	if grp.parent == nil {
		grp.c.Add(func() ServerShutdown { return shut }, nil)
	}
	return grp
}

func newGroup(name string, parent *group) *group {
	var pc *di.Container
	var pctx *srvCtx
	if parent != nil {
		pc = parent.c
		pctx = parent.ctx
	}
	log := zlog.New(name)
	c := di.New(pc, ctxType, shutType)
	ctx := newContext(pctx, log)
	grp := &group{
		name:        name,
		parent:      parent,
		children:    map[string]*group{},
		c:           c,
		ctx:         ctx,
		configHooks: []ConfigHook{},
		startHooks:  []StartHook{},
		stopHooks:   []StopHook{},
		healthHooks: []HealthHook{},
	}

	// Provide the Context, Shutdown per group
	grp.c.Add(func() Context { return grp.ctx }, nil)
	grp.c.Add(func() Shutdown { return grp.ctx.Shutdown }, nil)

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
	// g.c.add will call this function for each value produced by ctr
	// constructor method we then check if the produced value implements
	// any of the lifecycle hooks and cache them so that we can invoke them
	// as part of the server lifecycle.
	vf := func(v reflect.Value) error {
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
	// add the component constructor to the container
	return g.c.Add(ctr, vf)
}

// Invoke invokes a function with dependency injection.
func (g *group) Invoke(f interface{}) error {
	return g.c.Invoke(f, nil)
}

// Configure calls the configure hooks on all components registered for configuration.
func (g *group) Configure() error {
	g.ctx.Log().Info().Msg("configuring group")
	for _, h := range g.configHooks {
		if err := g.c.Invoke(h.Configure, nil); err != nil {
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
