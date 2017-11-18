package service

import (
	"reflect"

	"github.com/anuvu/cube/config"
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
	if parent == nil {
		c = newContainer(nil)
		ctx = newContext()
	} else {
		c = newContainer(parent.c)
		ctx = parent.ctx
	}
	grp := &Group{
		name:        name,
		parent:      parent,
		c:           c,
		ctx:         ctx,
		configHooks: []ConfigHook{},
		startHooks:  []StartHook{},
		stopHooks:   []StopHook{},
		healthHooks: []HealthHook{},
	}

	// Provide the context if we are the root group
	if grp.parent == nil {
		grp.c.add(func() Context {
			return grp.ctx
		})
	}
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
	return g.c.addWithProcessValue(ctr, vf)
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
	return nil
}

// Stop calls the stop hooks on all services registered for shutdown.
func (g *Group) Stop() error {
	var e error
	// Invoke the stop hooks in the reverse dependency order
	for _, h := range g.stopHooks {
		if err := g.c.invoke(h.Stop); err != nil {
			// FIXME: We need to make this multi-error
			e = err
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
	return true
}
