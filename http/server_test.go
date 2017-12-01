package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/anuvu/zlog"

	"github.com/anuvu/cube/component"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	port = 8989
	msg  = "hello"
)

type testHandler struct{}

func (th testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, err := w.Write([]byte(msg))
	if err != nil {
		panic(err)
	}
}

func TestHTTPServer(t *testing.T) {
	Convey("http server actually serves stuff", t, func() {
		ctx := component.RootContext(zlog.New("http.test"))
		s := New(ctx)
		So(s.(component.ConfigHook), ShouldNotBeNil)
		So(s.(component.StartHook), ShouldNotBeNil)
		So(s.(component.StopHook), ShouldNotBeNil)
		So(s.(component.HealthHook), ShouldNotBeNil)
		srv := s.(*server)
		cfg := srv.Config().(*configuration)
		cfg.Port = port
		So(srv.Configure(ctx), ShouldBeNil)
		So(srv.Start(ctx), ShouldBeNil)

		s.Register("/foo", testHandler{})

		// Write client to test the server
		So(srv.IsHealthy(ctx), ShouldBeTrue)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/foo", port))
		So(err, ShouldBeNil)
		bytes, err := ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)
		So(string(bytes), ShouldEqual, string(msg))

		// Stop the group
		So(srv.Stop(ctx), ShouldBeNil)
		So(srv.IsHealthy(ctx), ShouldBeFalse)
	})
}

func TestBadPort(t *testing.T) {
	Convey("http server with bad port", t, func() {
		ctx := component.RootContext(zlog.New("http.test"))
		s := New(ctx).(*server)
		cfg := s.Config().(*configuration)
		cfg.Port = -1
		So(s.Configure(ctx), ShouldBeNil)
		So(s.Start(ctx), ShouldNotBeNil)
	})
}
