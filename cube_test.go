package cube

import (
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/anuvu/cube/config"
	"github.com/anuvu/cube/service"
	. "github.com/smartystreets/goconvey/convey"
)

func newConfigStore(ctx service.Context) config.Store {
	ctx.Log().Info().Msg("tester config store created")
	r := strings.NewReader("")
	return config.NewJSONStore(r)
}

type tester struct {
}

func newtest(ctx service.Context) *tester {
	ctx.Log().Info().Msg("tester object created")
	return &tester{}
}

func (d *tester) Configure(ctx service.Context, store config.Store) error {
	return nil
}

func (d *tester) Start(ctx service.Context) error {
	return errors.New("bad start")
}

type stoptester struct {
}

func (d *stoptester) Stop(ctx service.Context) error {
	return errors.New("bad stop")
}

func TestCubePanics(t *testing.T) {
	Convey("cube main should panic if there is no config store", t, func() {
		initFunc := func(g ComponentGroup) { g.AddComponent(newtest) }
		So(func() { Main(initFunc) }, ShouldPanic)
	})
	Convey("cube main should panic dependencies are not met", t, func() {
		initFunc := func(g ComponentGroup) { g.AddComponent(func(i *int) {}) }
		So(func() { Main(initFunc) }, ShouldPanic)
	})
	Convey("cube main should panic on start errors", t, func() {
		initFunc := func(g ComponentGroup) {
			g.AddComponent(newConfigStore)
			g.AddComponent(newtest)
		}
		So(func() { Main(initFunc) }, ShouldPanic)
	})
	Convey("cube main should panic on stop errors", t, func() {
		initFunc := func(g ComponentGroup) {
			g.AddComponent(newConfigStore)
			g.AddComponent(func() *stoptester { return &stoptester{} })
			g.AddComponent(func(s *stoptester, k service.ServerShutdown) { k() })
		}
		So(func() { Main(initFunc) }, ShouldPanic)
	})
	Convey("calling shutdown handler should stop server", t, func() {
		initFunc := func(g ComponentGroup) {
			g.AddComponent(newConfigStore)
			g.AddComponent(func(s *shutDownHandler) {
				s.shut(syscall.SIGTERM)
			})
		}
		So(func() { Main(initFunc) }, ShouldNotPanic)
	})
}
