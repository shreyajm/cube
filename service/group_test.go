package service

import (
	"fmt"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type svc struct {
	startCalled     bool
	stopCalled      bool
	configureCalled bool
}

type svcWithHooks svc

func newSvcWithHooks(ctx Context) *svcWithHooks {
	s := &svcWithHooks{}
	ctx.AddLifecycle(&Lifecycle{
		ConfigHook: func(svc *svcWithHooks) { svc.configureCalled = true },
		StartHook:  func(svc *svcWithHooks) { svc.startCalled = true },
		StopHook:   func(svc *svcWithHooks) { svc.stopCalled = true },
		HealthHook: func() bool { return s.startCalled && !s.stopCalled },
	})
	return s
}

type svcWithErrors svc

func newSvcWithErrors(ctx Context) *svcWithErrors {
	s := &svcWithErrors{}
	ctx.AddLifecycle(&Lifecycle{
		ConfigHook: func(svc *svcWithErrors) error {
			svc.configureCalled = true
			return fmt.Errorf("config error")
		},
		StartHook: func(svc *svcWithErrors) error {
			svc.startCalled = true
			return fmt.Errorf("start error")
		},
		StopHook: func(svc *svcWithErrors) error {
			svc.stopCalled = true
			return fmt.Errorf("stop error")
		},
		HealthHook: func() bool { return s.startCalled && !s.stopCalled },
	})
	return s
}

func TestGroup(t *testing.T) {
	Convey("After we create a group", t, func() {
		grp := NewGroup("base", nil)
		So(grp, ShouldNotBeNil)
		So(grp.parent, ShouldBeNil)
		So(grp.ctx, ShouldNotBeNil)

		Convey("we should be able to add a service with no hooks", func() {
			s := &svc{}
			So(grp.AddService(func(ctx Context) *svc { return s }), ShouldBeNil)
			So(grp.Configure(), ShouldBeNil)
			So(grp.Start(), ShouldBeNil)
			So(grp.Stop(), ShouldBeNil)

			// Assert that none of the hooks are called
			So(s.configureCalled, ShouldBeFalse)
			So(s.stopCalled, ShouldBeFalse)
			So(s.startCalled, ShouldBeFalse)
		})

		Convey("we should be able to add service with hooks", func() {
			s := newSvcWithHooks(grp.ctx)
			So(grp.AddService(func(ctx Context) *svcWithHooks {
				return s
			}), ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)

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

		Convey("check service with errors", func() {
			s := newSvcWithErrors(grp.ctx)
			So(grp.AddService(func(ctx Context) *svcWithErrors {
				return s
			}), ShouldBeNil)
			So(grp.IsHealthy(), ShouldBeFalse)
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
}

func TestGroupHierarchy(t *testing.T) {
	Convey("Create the root group", t, func() {
		root := NewGroup("root", nil)
		So(root, ShouldNotBeNil)
		Convey("create a sub group", func() {
			grp := NewGroup("test", root)
			So(grp, ShouldNotBeNil)
			Convey("we should be able to add service with hooks", func() {
				s := newSvcWithHooks(grp.ctx)
				So(grp.AddService(func(ctx Context) *svcWithHooks {
					return s
				}), ShouldBeNil)
				So(grp.IsHealthy(), ShouldBeFalse)

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
	})
}
