package cube

import (
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/anuvu/cube/component"
	"github.com/anuvu/cube/config"
	. "github.com/smartystreets/goconvey/convey"
)

func newConfigStore(ctx component.Context) config.Store {
	ctx.Log().Info().Msg("tester config store created")
	r := strings.NewReader("")
	return config.NewJSONStore(r)
}

type tester struct {
}

func newtest(ctx component.Context) *tester {
	ctx.Log().Info().Msg("tester object created")
	return &tester{}
}

func (d *tester) Configure(ctx component.Context, store config.Store) error {
	return nil
}

func (d *tester) Start(ctx component.Context) error {
	return errors.New("bad start")
}

type stoptester struct {
}

func (d *stoptester) Stop(ctx component.Context) error {
	return errors.New("bad stop")
}

func TestCubePanics(t *testing.T) {
	Convey("cube main should panic if there is no config store", t, func() {
		initFunc := func(g component.Group) error { return g.Add(newtest) }
		So(func() { Main(initFunc) }, ShouldPanic)
	})

	Convey("cube main should panic dependencies are not met", t, func() {
		initFunc := func(g component.Group) error { return g.Add(func(i *int) {}) }
		So(func() { Main(initFunc) }, ShouldPanic)
	})

	Convey("cube main should panic on start errors", t, func() {
		initFunc := func(g component.Group) error {
			g.Add(newConfigStore)
			g.Add(newtest)
			return nil
		}
		So(func() { Main(initFunc) }, ShouldPanic)
	})
	Convey("cube main should panic on stop errors", t, func() {
		initFunc := func(g component.Group) error {
			g.Add(newConfigStore)
			g.Add(func() *stoptester { return &stoptester{} })
			g.Add(func(s *stoptester, k component.ServerShutdown) { k() })
			return nil
		}
		So(func() { Main(initFunc) }, ShouldPanic)
	})
	Convey("calling shutdown handler should stop server", t, func() {
		initFunc := func(g component.Group) error {
			g.Add(newConfigStore)
			g.Add(func(s *shutDownHandler) {
				s.shut(syscall.SIGTERM)
			})
			return nil
		}
		So(func() { Main(initFunc) }, ShouldNotPanic)
	})
}
