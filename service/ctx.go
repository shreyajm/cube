package service

import (
	"context"
)

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
	}
}

type srvCtx struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func (sc *srvCtx) Ctx() context.Context {
	return sc.ctx
}

func (sc *srvCtx) Shutdown() {
	sc.cancelFunc()
}
