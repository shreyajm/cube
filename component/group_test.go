package component

import (
	"fmt"
	"os"
	"testing"

	"github.com/anuvu/cube/config"
	. "github.com/smartystreets/goconvey/convey"
)

type cmp struct {
	startCalled     bool
	stopCalled      bool
	configureCalled bool
	configCalled    bool
	errorConfig     bool
}

type cmpWithHooks cmp

func newCmpConfigError() *cmpWithHooks {
	return &cmpWithHooks{errorConfig: true}
}

func newCmpWithHooks(ctx Context) *cmpWithHooks {
	return &cmpWithHooks{}
}

func (cmp *cmpWithHooks) Config() config.Config {
	cmp.configCalled = true
	if cmp.errorConfig {
		return &config.BaseConfig{ConfigKey: "hooks"}
	}
	return nil
}

func (cmp *cmpWithHooks) Configure(ctx Context) error {
	cmp.configureCalled = true
	return nil
}
func (cmp *cmpWithHooks) Start(ctx Context) error {
	cmp.startCalled = true
	return nil
}
func (cmp *cmpWithHooks) Stop(ctx Context) error {
	cmp.stopCalled = true
	return nil
}

func (cmp *cmpWithHooks) IsHealthy(ctx Context) bool { return cmp.startCalled && !cmp.stopCalled }

type cmpWithErrors cmp

func newCmpWithErrors(ctx Context) *cmpWithErrors {
	return &cmpWithErrors{}
}

func (cmp *cmpWithErrors) Config() config.Config {
	cmp.configCalled = true
	return nil
}

func (cmp *cmpWithErrors) Configure(ctx Context) error {
	cmp.configureCalled = true
	return fmt.Errorf("config error")
}
func (cmp *cmpWithErrors) Start(ctx Context) error {
	cmp.startCalled = true
	return fmt.Errorf("config error")
}
func (cmp *cmpWithErrors) Stop(ctx Context) error {
	cmp.stopCalled = true
	return fmt.Errorf("config error")
}

func (cmp *cmpWithErrors) IsHealthy(ctx Context) bool { return cmp.startCalled && !cmp.stopCalled }

func TestGroup(t *testing.T) {
	// Replace os.Args
	oldArgs := os.Args
	os.Args = []string{"group.test"}
	defer func() { os.Args = oldArgs }()
	Convey("After we create a group", t, func() {
		grp := New("base").(*group)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)

		Convey("we should be able to add a component with no hooks", func() {
			So(grp.Add(func(ctx Context) *cmp { return &cmp{} }), ShouldBeNil)
			So(grp.Configure(), ShouldBeNil)
			So(grp.Start(), ShouldBeNil)
			So(grp.Stop(), ShouldBeNil)

			// Assert that none of the hooks are called
			grp.Invoke(func(s *cmp) {
				So(s.configureCalled, ShouldBeFalse)
				So(s.stopCalled, ShouldBeFalse)
				So(s.startCalled, ShouldBeFalse)
			})
		})

		Convey("we should be able to add component with hooks", func() {
			err := grp.Add(newCmpWithHooks)
			So(err, ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)

			grp.Invoke(func(s *cmpWithHooks) {
				Convey("we should be able to configure the group", func() {
					So(grp.Configure(), ShouldBeNil)
					So(s.configureCalled, ShouldBeTrue)
					So(grp.IsHealthy(), ShouldBeFalse)
				})
				Convey("we should be able to start the group", func() {
					So(grp.Start(), ShouldBeNil)
					So(s.startCalled, ShouldBeTrue)
					So(grp.IsHealthy(), ShouldBeTrue)
				})
				Convey("we should be able to stop the group", func() {
					So(grp.Stop(), ShouldBeNil)
					So(s.stopCalled, ShouldBeTrue)
					So(grp.IsHealthy(), ShouldBeFalse)
				})
			})
		})

		Convey("check component with errors", func() {
			So(grp.Add(newCmpWithErrors), ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)
			grp.Invoke(func(s *cmpWithErrors) {
				Convey("configure the group should be error", func() {
					So(grp.Configure(), ShouldNotBeNil)
					So(s.configureCalled, ShouldBeTrue)
					So(grp.IsHealthy(), ShouldBeFalse)
				})
				Convey("start should be error", func() {
					So(grp.Start(), ShouldNotBeNil)
					So(s.startCalled, ShouldBeTrue)
					So(s.stopCalled, ShouldBeTrue)
					So(grp.IsHealthy(), ShouldBeFalse)
				})
				Convey("stop should be error", func() {
					So(grp.Stop(), ShouldNotBeNil)
					So(s.stopCalled, ShouldBeTrue)
					So(grp.IsHealthy(), ShouldBeFalse)
				})
			})
		})
	})
}

func TestGroupHierarchy(t *testing.T) {
	// Replace os.Args
	oldArgs := os.Args
	os.Args = []string{"group.test"}
	defer func() { os.Args = oldArgs }()

	Convey("Create the root group", t, func() {
		root := New("root").(*group)
		So(root, ShouldNotBeNil)
		grp := root.New("test").(*group)
		So(grp, ShouldNotBeNil)
		Convey("we should be able to add component with hooks", func() {
			So(grp.Add(newCmpWithHooks), ShouldBeNil)
			So(root.IsHealthy(), ShouldBeFalse)

			grp.Invoke(func(s *cmpWithHooks) {
				Convey("we should be able to configure the group", func() {
					So(root.Configure(), ShouldBeNil)
					So(s.configureCalled, ShouldBeTrue)
					So(root.IsHealthy(), ShouldBeFalse)
				})
				Convey("we should be able to start the group", func() {
					So(root.Start(), ShouldBeNil)
					So(s.startCalled, ShouldBeTrue)
					So(root.IsHealthy(), ShouldBeTrue)
				})
				Convey("we should be able to stop the group", func() {
					So(root.Stop(), ShouldBeNil)
					So(s.stopCalled, ShouldBeTrue)
					So(root.IsHealthy(), ShouldBeFalse)
				})
			})
		})
		Convey("check component with errors", func() {
			So(grp.Add(newCmpWithErrors), ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)
			grp.Invoke(func(s *cmpWithErrors) {
				Convey("configure the group should be error", func() {
					So(root.Configure(), ShouldNotBeNil)
					So(s.configureCalled, ShouldBeTrue)
					So(root.IsHealthy(), ShouldBeFalse)
				})
				Convey("start should be error", func() {
					So(root.Start(), ShouldNotBeNil)
					So(s.startCalled, ShouldBeTrue)
					So(s.stopCalled, ShouldBeTrue)
					So(root.IsHealthy(), ShouldBeFalse)
				})
				Convey("stop should be error", func() {
					So(root.Stop(), ShouldNotBeNil)
					So(s.stopCalled, ShouldBeTrue)
					So(root.IsHealthy(), ShouldBeFalse)
				})
			})
		})
		Convey("check for unique contexts", func() {
			var baseCtx Context
			var derivedCtx Context
			// Capture the base context
			root.Invoke(func(ctx Context) {
				baseCtx = ctx
				So(ctx, ShouldNotBeNil)
			})

			// Capture the derived context
			grp.Invoke(func(ctx Context) {
				derivedCtx = ctx
				So(ctx, ShouldNotBeNil)
			})

			// Assert that base and derived contexts are not equal
			So(baseCtx, ShouldNotEqual, derivedCtx)
		})
	})
}

func TestBadFileStore(t *testing.T) {
	// Replace os.Args
	oldArgs := os.Args
	os.Args = []string{"group.test", "--config.file", "bad_file_name"}
	defer func() { os.Args = oldArgs }()
	Convey("Create the root group", t, func() {
		grp := New("base").(*group)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)
		So(grp.Configure(), ShouldNotBeNil)
		So(grp.store.Get(&config.BaseConfig{ConfigKey: "test"}), ShouldNotBeNil)
	})
}

func TestFileStore(t *testing.T) {
	// Replace os.Args
	oldArgs := os.Args
	os.Args = []string{"group.test", "--config.file", "./cfg_test.json"}
	defer func() { os.Args = oldArgs }()
	Convey("Create the root group", t, func() {
		grp := New("base").(*group)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)
		So(grp.Configure(), ShouldBeNil)
		So(grp.store.Get(&config.BaseConfig{ConfigKey: "test"}), ShouldNotBeNil)
	})
}

func TestMemStore(t *testing.T) {
	// Replace os.Args
	oldArgs := os.Args
	os.Args = []string{"group.test", "--config.mem", "{}"}
	defer func() { os.Args = oldArgs }()
	Convey("Create the root group", t, func() {
		grp := New("base").(*group)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)
		So(grp.Configure(), ShouldBeNil)
		So(grp.store.Get(&config.BaseConfig{ConfigKey: "test"}), ShouldNotBeNil)
		grp.Add(newCmpConfigError)
		So(grp.Configure(), ShouldNotBeNil)
	})
}

func TestBadCli(t *testing.T) {
	// Replace os.Args
	oldArgs := os.Args
	os.Args = []string{"group.test", "--config.memx", "{}"}
	defer func() { os.Args = oldArgs }()
	Convey("Create the root group", t, func() {
		grp := New("base").(*group)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)
		So(grp.Configure(), ShouldNotBeNil)
		So(grp.store.Get(&config.BaseConfig{ConfigKey: "test"}), ShouldNotBeNil)
	})
}
