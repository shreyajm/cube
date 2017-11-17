package service

import "reflect"

// Group is a group of services, that have inter-dependencies.
type Group struct {
	name     string
	parent   *Group
	c        *container
	ctx      *srvCtx
	invokers []interface{}
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
		name:     name,
		parent:   parent,
		c:        c,
		ctx:      ctx,
		invokers: []interface{}{},
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
	for _, h := range g.ctx.hooks {
		if h.ConfigHook != nil {
			if err := g.c.invoke(h.ConfigHook); err != nil {
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
			if err := g.c.invoke(h.StartHook); err != nil {
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
			if err := g.c.invoke(h.StopHook); err != nil {
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
