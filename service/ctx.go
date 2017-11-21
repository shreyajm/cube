package service

import (
	"context"

	"github.com/anuvu/zlog"
)

// Context provides a wrapper interface for go context and logger.
//
// Ctx() returns the underlying go context.
//
// Log() returns the group's logger
type Context interface {
	Ctx() context.Context
	Log() zlog.Logger
}

// Shutdown invokes the shutdown sequence
type Shutdown func()

func newContext(p *srvCtx, log zlog.Logger) *srvCtx {
	c := context.Background()
	if p != nil {
		c = p.ctx
	}
	ctx, cancelFunc := context.WithCancel(c)
	return &srvCtx{
		ctx:        ctx,
		cancelFunc: cancelFunc,
		log:        log,
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
