package service

import (
	"reflect"

	"github.com/anuvu/cube/config"
	"github.com/anuvu/zlog"
)

// ConfigHook is the interface that provides the configuration call back for the service.
type ConfigHook interface {
	Configure(ctx Context, store config.Store) error
}

// StartHook is the interface that provides the start call back for the service.
type StartHook interface {
	Start(ctx Context) error
}

// StopHook is the interface that provides the stop call back for the service.
type StopHook interface {
	Stop(ctx Context) error
}

// HealthHook is the interface that provides the health call back for the service.
type HealthHook interface {
	IsHealthy(ctx Context) bool
}

// Group is a group of services, that have inter-dependencies.
type Group struct {
	name        string
	parent      *Group
	children    map[string]*Group
	c           *container
	ctx         *srvCtx
	configHooks []ConfigHook
	startHooks  []StartHook
	stopHooks   []StopHook
	healthHooks []HealthHook
}

// NewGroup creates a new service group with the specified parent. If the parent is nil
// this group is the root group.
func NewGroup(name string, parent *Group) *Group {
	var c *container
	var ctx *srvCtx
	log := zlog.New(name)
	if parent == nil {
		c = newContainer(nil)
		ctx = newContext(nil, log)
	} else {
		c = newContainer(parent.c)
		ctx = newContext(parent.ctx, log)
	}
	grp := &Group{
		name:        name,
		parent:      parent,
		children:    map[string]*Group{},
		c:           c,
		ctx:         ctx,
		configHooks: []ConfigHook{},
		startHooks:  []StartHook{},
		stopHooks:   []StopHook{},
		healthHooks: []HealthHook{},
	}
	if parent != nil {
		// FIXME: Potential child name collision, check for it.
		parent.children[name] = grp
	}

	// Provide the context and shutdown func
	grp.c.add(func() Context { return grp.ctx }, nil)
	grp.c.add(func() Shutdown { return grp.ctx.Shutdown }, nil)

	return grp
}

// AddService adds a new service constructor to the service group.
func (g *Group) AddService(ctr interface{}) error {
	vf := func(v reflect.Value) {
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
	}
	// add the service constructor to the container
	return g.c.add(ctr, vf)
}

// Invoke invokes a function with dependency injection.
func (g *Group) Invoke(f interface{}) error {
	return g.c.invoke(f)
}

// Configure calls the configure hooks on all services registered for configuration.
func (g *Group) Configure() error {
	for _, h := range g.configHooks {
		if err := g.c.invoke(h.Configure); err != nil {
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

// Start calls the start hooks on all services registered for startup.
// If an error occurs on any hook, subsequent start calls are abandoned
// and a best effort stop is initiated.
func (g *Group) Start() error {
	for _, h := range g.startHooks {
		if err := g.c.invoke(h.Start); err != nil {
			// We need to call all stop hooks and ignore errors
			// as we dont know which services are actually participating
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

// Stop calls the stop hooks on all services registered for shutdown.
func (g *Group) Stop() error {
	var e error

	// Stop all the child groups first
	for _, child := range g.children {
		e = child.Stop()
	}

	// Signal all async services in this group to stop
	g.ctx.Shutdown()

	// Invoke the stop hooks in the reverse dependency order
	if len(g.stopHooks) > 0 {
		for i := len(g.stopHooks) - 1; i == 0; i-- {
			h := g.stopHooks[i]
			if err := g.c.invoke(h.Stop); err != nil {
				// FIXME: We need to make this multi-error
				e = err
			}
		}
	}
	return e
}

// IsHealthy returns true if all services health hooks return true else false
func (g *Group) IsHealthy() bool {
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
