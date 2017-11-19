package service

import (
	"context"

	"github.com/anuvu/zlog"
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
	Log() zlog.Logger
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
		log:        zlog.New("cube"),
	}
}

type srvCtx struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
	log        zlog.Logger
}

func (sc *srvCtx) Ctx() context.Context {
	return sc.ctx
}

func (sc *srvCtx) Shutdown() {
	sc.cancelFunc()
}

func (sc *srvCtx) Log() zlog.Logger {
	return sc.log
}
