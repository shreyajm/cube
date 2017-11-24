package di

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
		c := New(nil)
		Convey("cannot add non-func or nil", func() {
			So(c.Add(10, nil), ShouldNotBeNil)
			So(c.Add(nil, nil), ShouldNotBeNil)
		})
		Convey("cannot add a func in case of error", func() {
			So(c.Add(func() (*testS1, error) { return nil, errors.New("test error") }, nil), ShouldNotBeNil)
		})
		Convey("should accept a variadic constructor but ignore the variadic list", func() {
			So(c.Add(func(b ...int) {}, nil), ShouldBeNil)
		})
		Convey("can add a constructor", func() {
			So(c.Add(func() *testS1 { return &testS1{} }, nil), ShouldBeNil)
			So(c.Invoke(func(t *testS1) {}, nil), ShouldBeNil)
			So(c.Add(func() *testS1 { return &testS1{} }, nil), ShouldNotBeNil)
		})
		Convey("cannot add constructor with bad dependencies", func() {
			So(c.Add(func(c *Container) *Container { return nil }, nil), ShouldNotBeNil)
		})
		Convey("add with a value processor", func() {
			e := c.Add(
				func() *testS1 { return &testS1{} },
				func(v reflect.Value) error {
					So(v.Interface(), ShouldNotBeNil)
					return nil
				})
			So(e, ShouldBeNil)
			e = c.Add(
				func() *testS2 { return &testS2{} },
				func(v reflect.Value) error {
					return errors.New("test")
				})
			So(e, ShouldNotBeNil)
		})
		Convey("can create a hierarchy", func() {
			cc := New(c)
			So(cc, ShouldNotBeNil)
			So(c.Add(func() *testS1 { return &testS1{} }, nil), ShouldBeNil)
			So(cc.Add(func(s1 *testS1) *testS2 { return &testS2{} }, nil), ShouldBeNil)
			So(cc.Invoke(func(s2 *testS2) {}, nil), ShouldBeNil)
			dupC := New(cc, reflect.TypeOf(&testS2{}).Elem())

			// Allowed to add have dup for parent, but not in the same container
			So(dupC.Add(func(s1 *testS1) *testS2 { return &testS2{} }, nil), ShouldBeNil)
			So(dupC.Add(func(s1 *testS1) *testS2 { return &testS2{} }, nil), ShouldNotBeNil)
		})
	})
}
