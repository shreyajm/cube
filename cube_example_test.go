package cube_test

import (
	"strings"
	"time"

	"github.com/anuvu/cube"
	"github.com/anuvu/cube/component"
	"github.com/anuvu/cube/config"
)

func newConfigStore(ctx component.Context) config.Store {
	ctx.Log().Info().Msg("dummy config store created")
	r := strings.NewReader("")
	return config.NewJSONStore(r)
}

type dummy struct {
}

func newDummy(ctx component.Context) *dummy {
	ctx.Log().Info().Msg("dummy object created")
	return &dummy{}
}

func (d *dummy) Configure(ctx component.Context, store config.Store) error {
	ctx.Log().Info().Msg("dummy object configured")
	return nil
}

func (d *dummy) Start(ctx component.Context) error {
	ctx.Log().Info().Msg("dummy object started")
	return nil
}

func (d *dummy) Stop(ctx component.Context) error {
	ctx.Log().Info().Msg("dummy object stopped")
	return nil
}

type killer struct {
	kill component.ServerShutdown
}

func newKiller(d *dummy, k component.ServerShutdown, ctx component.Context) *killer {
	ctx.Log().Info().Msg("killer object created")
	// Make a dummy dependency so that this will start after dummy is started
	return &killer{k}
}

func (k *killer) Start(ctx component.Context) error {
	go func() {
		// Wait for a second and initiate a shutdown
		time.Sleep(time.Millisecond)
		ctx.Log().Info().Msg("Killing the server")
		k.kill()
	}()
	return nil
}

func ExampleMain() {
	cube.Main(func(g component.Group) error {
		g.Add(newConfigStore)
		g.Add(newDummy)
		g.Add(newKiller)
		return nil
	})

	// Output:
	// {"level":"info","name":"cube.test","message":"dummy config store created"}
	// {"level":"info","name":"cube.test","message":"dummy object created"}
	// {"level":"info","name":"cube.test","message":"killer object created"}
	// {"level":"info","name":"cube.test-core","message":"configuring group"}
	// {"level":"info","name":"cube.test","message":"configuring group"}
	// {"level":"info","name":"cube.test","message":"dummy object configured"}
	// {"level":"info","name":"cube.test-core","message":"starting group"}
	// {"level":"info","name":"cube.test","message":"starting group"}
	// {"level":"info","name":"cube.test","message":"dummy object started"}
	// {"level":"info","name":"cube.test","message":"Killing the server"}
	// {"level":"info","name":"cube.test","message":"stopping group"}
	// {"level":"info","name":"cube.test","message":"dummy object stopped"}
	// {"level":"info","name":"cube.test-core","message":"stopping group"}
}
