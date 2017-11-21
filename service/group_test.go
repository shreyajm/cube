package service

import (
	"fmt"
	"strings"
	"testing"

	"github.com/anuvu/cube/config"
	. "github.com/smartystreets/goconvey/convey"
)

func newConfigStore() config.Store {
	r := strings.NewReader(`{"http": {"port": 8080}}
		{"logger": {"file": "/var/log/test.log"}}`)
	return config.NewJSONStore(r)
}

type svc struct {
	startCalled     bool
	stopCalled      bool
	configureCalled bool
}

type svcWithHooks svc

func newSvcWithHooks(ctx Context) *svcWithHooks {
	return &svcWithHooks{false, false, false}
}

func (svc *svcWithHooks) Configure(ctx Context, store config.Store) error {
	svc.configureCalled = true
	return nil
}
func (svc *svcWithHooks) Start(ctx Context) error {
	svc.startCalled = true
	return nil
}
func (svc *svcWithHooks) Stop(ctx Context) error {
	svc.stopCalled = true
	return nil
}

func (svc *svcWithHooks) IsHealthy(ctx Context) bool { return svc.startCalled && !svc.stopCalled }

type svcWithErrors svc

func newSvcWithErrors(ctx Context) *svcWithErrors {
	return &svcWithErrors{}
}

func (svc *svcWithErrors) Configure(ctx Context, store config.Store) error {
	svc.configureCalled = true
	return fmt.Errorf("config error")
}
func (svc *svcWithErrors) Start(ctx Context) error {
	svc.startCalled = true
	return fmt.Errorf("config error")
}
func (svc *svcWithErrors) Stop(ctx Context) error {
	svc.stopCalled = true
	return fmt.Errorf("config error")
}

func (svc *svcWithErrors) IsHealthy(ctx Context) bool { return svc.startCalled && !svc.stopCalled }

func TestGroup(t *testing.T) {
	Convey("After we create a group", t, func() {
		grp := NewGroup("base", nil)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)
		So(grp.AddService(newConfigStore), ShouldBeNil)

		Convey("we should be able to add a service with no hooks", func() {
			So(grp.AddService(func(ctx Context) *svc { return &svc{} }), ShouldBeNil)
			So(grp.Configure(), ShouldBeNil)
			So(grp.Start(), ShouldBeNil)
			So(grp.Stop(), ShouldBeNil)

			// Assert that none of the hooks are called
			grp.Invoke(func(s *svc) {
				So(s.configureCalled, ShouldBeFalse)
				So(s.stopCalled, ShouldBeFalse)
				So(s.startCalled, ShouldBeFalse)
			})
		})

		Convey("we should be able to add service with hooks", func() {
			err := grp.AddService(newSvcWithHooks)
			So(err, ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)

			grp.Invoke(func(s *svcWithHooks) {
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

		Convey("check service with errors", func() {
			So(grp.AddService(newSvcWithErrors), ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)
			grp.Invoke(func(s *svcWithErrors) {
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
	Convey("Create the root group", t, func() {
		root := NewGroup("root", nil)
		So(root, ShouldNotBeNil)
		So(root.AddService(newConfigStore), ShouldBeNil)
		grp := NewGroup("test", root)
		So(grp, ShouldNotBeNil)
		Convey("we should be able to add service with hooks", func() {
			So(grp.AddService(newSvcWithHooks), ShouldBeNil)
			So(root.IsHealthy(), ShouldBeFalse)

			grp.Invoke(func(s *svcWithHooks) {
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
		Convey("check service with errors", func() {
			So(grp.AddService(newSvcWithErrors), ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)
			grp.Invoke(func(s *svcWithErrors) {
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
