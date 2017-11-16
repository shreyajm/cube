package service

import (
	"context"
)

// Lifecycle captures the service lifecycle hooks.
type Lifecycle struct {
	// Service configure hook, must be a function
	ConfigHook interface{}

	// Service Start hook, must be a function
	StartHook interface{}

	// Service Stop hook, must be a function
	StopHook interface{}

	// Service health hook
	HealthHook func() bool
}

// Context provides a wrapper interface for go context.
//
// Ctx() returns the underlying go context.
//
// Shutdown() cancels the go context.
//
// AddLifecycle() adds a service lifecycle hook to the context
type Context interface {
	Ctx() context.Context
	Shutdown()
	AddLifecycle(*Lifecycle)
}

// NewContext creates a new service context.
func NewContext() Context {
	return newContext()
}

func newContext() *srvCtx {
	c := context.Background()
	ctx, cancelFunc := context.WithCancel(c)
	return &srvCtx{
		ctx:        ctx,
		cancelFunc: cancelFunc,
		hooks:      []*Lifecycle{},
	}
}

type srvCtx struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	hooks      []*Lifecycle
}

func (sc *srvCtx) Ctx() context.Context {
	return sc.ctx
}

func (sc *srvCtx) Shutdown() {
	sc.cancelFunc()
}

func (sc *srvCtx) AddLifecycle(lc *Lifecycle) {
	sc.hooks = append(sc.hooks, lc)
}
