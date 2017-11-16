package service

import (
	"github.com/anuvu/dig"
)

// Group is a group of services, that have inter-dependencies.
type Group struct {
	name      string
	parent    *Group
	container *dig.Container
	ctx       *srvCtx
	invokers  []interface{}
}

// NewGroup creates a new service group with the specified parent. If the parent is nil
// this group is the root group.
func NewGroup(name string, parent *Group) *Group {
	var container *dig.Container
	var ctx *srvCtx
	if parent == nil {
		container = dig.New()
		ctx = newContext()
	} else {
		container = dig.NewWithParent(parent.container)
		ctx = parent.ctx
	}
	grp := &Group{
		name:      name,
		parent:    parent,
		container: container,
		ctx:       ctx,
		invokers:  []interface{}{},
	}

	// Provide the context if we are the root group
	if grp.parent == nil {
		grp.container.Provide(func() Context {
			return grp.ctx
		})
	}
	return grp
}

// AddService adds a new service constructor to the service group.
func (g *Group) AddService(ctr interface{}, invoker interface{}) error {
	if invoker != nil {
		g.invokers = append(g.invokers, invoker)
	}
	// add the service constructor to the container
	return g.container.Provide(ctr)
}

// Create creates the leaf level services, that bootstraps the service group
func (g *Group) Create() error {
	for _, inv := range g.invokers {
		if err := g.container.Invoke(inv); err != nil {
			return err
		}
	}

	return nil
}

// Invoke invokes a function with dependency injection.
func (g *Group) Invoke(f interface{}) error {
	return g.container.Invoke(f)
}

// Configure calls the configure hooks on all services registered for configuration.
func (g *Group) Configure() error {
	for _, h := range g.ctx.hooks {
		if h.ConfigHook != nil {
			if err := g.container.Invoke(h.ConfigHook); err != nil {
				return err
			}
		}
	}
	return nil
}

// Start calls the start hooks on all services registered for startup.
// If an error occurs on any hook, subsequent start calls are abandoned
// and a best effort stop is initiated.
func (g *Group) Start() error {
	for i, h := range g.ctx.hooks {
		if h.StartHook != nil {
			if err := g.container.Invoke(h.StartHook); err != nil {
				defer g.stop(i + 1)
				return err
			}
		}
	}
	return nil
}

// Stop calls the stop hooks on all services registered for shutdown.
func (g *Group) Stop() error {
	// Invoke the stop hooks in the reverse dependency order
	return g.stop(len(g.ctx.hooks))
}

func (g *Group) stop(index int) error {
	for i := index; i > 0; {
		i--
		h := g.ctx.hooks[i]
		if h.StopHook != nil {
			if err := g.container.Invoke(h.StopHook); err != nil {
				return err
			}
		}
	}
	return nil
}

// IsHealthy returns true if all services health hooks return true else false
func (g *Group) IsHealthy() bool {
	for _, h := range g.ctx.hooks {
		if h.HealthHook != nil && !h.HealthHook() {
			return false
		}
	}
	return true
}
