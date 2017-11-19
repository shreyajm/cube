package cli

import (
	"os"
	"strings"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/anuvu/cube/config"
	"github.com/anuvu/cube/service"
)

func newConfigStore() config.Store {
	r := strings.NewReader(`{"http": {"port": 8080}}
		{"logger": {"file": "/var/log/test.log"}}`)
	return config.NewJSONStore(r)
}

func TestCli(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"cli_test", "-foo", "bar"}

	Convey("After we add the CLI service, then invoke a function that requires a flagset", t, func() {
		grp := service.NewGroup("cli", nil)
		grp.AddService(newConfigStore)
		So(grp.AddService(New), ShouldBeNil)
		grp.Invoke(func(fs *Cli) {
			So(fs, ShouldNotBeNil)
			fs.Flags.String("foo", "baz", "usage")
			So(grp.Configure(), ShouldBeNil)
			So(fs.Flags.Parsed(), ShouldBeTrue)
			fooFlag := fs.Flags.Lookup("foo")
			So(fooFlag.Value.String(), ShouldEqual, "bar")
		})
	})
	os.Args = oldArgs
}
