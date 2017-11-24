package http

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/anuvu/cube/component"
	"github.com/anuvu/cube/config"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	port = 8989
	msg  = "hello"
)

func newConfigStore() config.Store {
	r := strings.NewReader(fmt.Sprintf(`{"http": {"port": %d}}`, port))
	return config.NewJSONStore(r)
}

func newBadConfig() config.Store {
	r := strings.NewReader(fmt.Sprintf(`{"httpx": {"portd": %d}}`, port))
	return config.NewJSONStore(r)
}
func newBadPort() config.Store {
	r := strings.NewReader(`{"http": {"port": -1}}`)
	return config.NewJSONStore(r)
}

type testHandler struct{}

func (th testHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	_, err := w.Write([]byte(msg))
	if err != nil {
		panic(err)
	}
}

func TestHTTPServer(t *testing.T) {
	Convey("http server actually serves stuff", t, func() {
		grp := component.New("http")
		So(grp.Add(newConfigStore), ShouldBeNil)
		So(grp.Invoke(func(s config.Store) error {
			return s.Open()
		}), ShouldBeNil)
		So(grp.Add(New), ShouldBeNil)
		So(grp.Configure(), ShouldBeNil)
		So(grp.Start(), ShouldBeNil)

		grp.Invoke(func(s Server) {
			s.Register("/foo", testHandler{})
		})

		// Write client to test the server
		So(grp.IsHealthy(), ShouldBeTrue)
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/foo", port))
		So(err, ShouldBeNil)
		bytes, err := ioutil.ReadAll(resp.Body)
		So(err, ShouldBeNil)
		So(string(bytes), ShouldEqual, string(msg))

		// Stop the group
		So(grp.Stop(), ShouldBeNil)
		So(grp.IsHealthy(), ShouldBeFalse)
	})
}

func TestBadConfig(t *testing.T) {
	Convey("http server with bad config", t, func() {
		grp := component.New("http")
		So(grp.Add(newBadConfig), ShouldBeNil)
		So(grp.Invoke(func(s config.Store) error {
			return s.Open()
		}), ShouldBeNil)
		So(grp.Add(New), ShouldBeNil)
		So(grp.Configure(), ShouldNotBeNil)
	})
}

func TestBadPort(t *testing.T) {
	Convey("http server with bad port", t, func() {
		grp := component.New("http")
		So(grp.Add(newBadPort), ShouldBeNil)
		So(grp.Invoke(
			func(s config.Store) error {
				return s.Open()
			}), ShouldBeNil)
		So(grp.Add(New), ShouldBeNil)
		So(grp.Configure(), ShouldBeNil)
		So(grp.Start(), ShouldNotBeNil)
	})
}
