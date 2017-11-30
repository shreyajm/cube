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

type testS3 struct {
}

func TestContainer(t *testing.T) {
	Convey("Create a container", t, func() {
		c := New(nil)
		Convey("cannot add non-func or nil", func() {
			So(c.Add(10), ShouldBeError)
			So(c.Add(nil), ShouldBeError)
			So(c.Invoke(nil, nil), ShouldBeError)
		})
		Convey("can add a func in case of error but cant create", func() {
			So(c.Add(func() (*testS1, error) { return nil, errors.New("test error") }), ShouldBeNil)
			So(c.Create(nil), ShouldBeError)
		})
		Convey("should accept a variadic constructor but ignore the variadic list", func() {
			So(c.Add(func(b ...int) {}), ShouldBeError)
			So(c.Add(func(b ...int) int { return 0 }), ShouldBeNil)
		})
		Convey("can add a constructor", func() {
			So(c.Add(func() *testS1 { return &testS1{} }), ShouldBeNil)
			So(c.Add(func() *testS1 { return &testS1{} }), ShouldBeError)
			So(c.Create(nil), ShouldBeNil)
			So(c.Invoke(func(t *testS1) {}, nil), ShouldBeNil)
		})
		Convey("cannot create constructor with bad dependencies", func() {
			So(c.Add(func(c *Container) *Container { return nil }), ShouldBeError)
			So(c.Add(func(e error) int { return 0 }), ShouldBeError)
			So(c.Add(func(c *Container) error { return errors.New("error") }), ShouldBeError)
			So(c.Add(func(c *Container) int { return 0 }), ShouldBeNil)
			So(c.Create(nil), ShouldBeError)
		})
		Convey("create with a value processor", func() {
			So(c.Add(func() *testS1 { return &testS1{} }), ShouldBeNil)
			e := c.Create(func(v reflect.Value) error {
				So(v.Interface(), ShouldNotBeNil)
				return nil
			})
			So(e, ShouldBeNil)
		})
		Convey("can add constructors out of order and still construct", func() {
			So(c.Add(func(*testS1) int { return 0 }), ShouldBeNil)
			So(c.Add(func() *testS1 { return &testS1{} }), ShouldBeNil)
			So(c.Create(nil), ShouldBeNil)
		})
		Convey("create with a error value processor", func() {
			So(c.Add(func() *testS1 { return &testS1{} }), ShouldBeNil)
			e := c.Create(
				func(v reflect.Value) error {
					return errors.New("test")
				})
			So(e, ShouldBeError)
		})
		Convey("can create a hierarchy", func() {
			cc := New(c)
			So(cc, ShouldNotBeNil)
			So(c.Add(func() *testS1 { return &testS1{} }), ShouldBeNil)
			So(cc.Add(func(s1 *testS1) *testS2 { return &testS2{} }), ShouldBeNil)
			So(c.Create(nil), ShouldBeNil)
			So(cc.Create(nil), ShouldBeNil)
			So(cc.Invoke(func(s2 *testS2) {}, nil), ShouldBeNil)
			dupC := New(cc, reflect.TypeOf(&testS2{}).Elem())

			// Allowed to add have dup for parent, but not in the same container
			So(dupC.Add(func(s1 *testS1) *testS2 { return &testS2{} }), ShouldBeNil)
			So(dupC.Add(func(s1 *testS1) (*testS2, *testS3) { return &testS2{}, &testS3{} }), ShouldBeError)
			So(dupC.Add(func(s1 *testS1) (*testS3, *testS2) { return &testS3{}, &testS2{} }), ShouldBeError)
			So(dupC.Add(func(s1 *testS1) *testS2 { return &testS2{} }), ShouldBeError)
			So(dupC.Create(nil), ShouldBeNil)

			// Check for bad duplicates across container ancestry
			badC := New(c)
			So(badC.Add(func(s1 *testS1) *testS2 { return &testS2{} }), ShouldBeNil)
			So(badC.Add(func() *testS1 { return &testS1{} }), ShouldBeNil)
			So(badC.Create(nil), ShouldBeError)
		})
	})
}
