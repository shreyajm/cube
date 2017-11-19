package service

import (
	"errors"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

type testS1 struct {
}

type testS2 struct {
}

func TestContainer(t *testing.T) {
	Convey("Create a container", t, func() {
		c := newContainer(nil)
		Convey("cannot add non-func or nil", func() {
			So(c.add(10, nil), ShouldNotBeNil)
			So(c.add(nil, nil), ShouldNotBeNil)
		})
		Convey("cannot add a func in case of error", func() {
			So(c.add(func() (*testS1, error) { return nil, errors.New("test error") }, nil), ShouldNotBeNil)
		})
		Convey("should accept a variadic constructor but ignore the variadic list", func() {
			So(c.add(func(b ...int) {}, nil), ShouldBeNil)
		})
		Convey("can add a constructor", func() {
			So(c.add(func() *testS1 { return &testS1{} }, nil), ShouldBeNil)
			So(c.invoke(func(t *testS1) {}), ShouldBeNil)
			So(c.add(func() *testS1 { return &testS1{} }, nil), ShouldNotBeNil)
		})
		Convey("cannot add constructor with bad dependencies", func() {
			So(c.add(func(c *container) *container { return nil }, nil), ShouldNotBeNil)
		})
		Convey("add with a value processor", func() {
			e := c.add(
				func() *testS1 { return &testS1{} },
				func(v reflect.Value) {
					So(v.Interface(), ShouldNotBeNil)
				})
			So(e, ShouldBeNil)
		})
		Convey("can create a hierarchy", func() {
			cc := newContainer(c)
			So(cc, ShouldNotBeNil)
			c.add(func() *testS1 { return &testS1{} }, nil)
			So(cc.add(func(s1 *testS1) *testS2 { return &testS2{} }, nil), ShouldBeNil)
			So(cc.invoke(func(s2 *testS2) {}), ShouldBeNil)
		})
	})
}
