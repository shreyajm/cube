package cli

import (
	"flag"

	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"github.com/anuvu/cube/service"
)

func TestCli(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"cli_test", "-foo", "bar"}

	Convey("After we add the CLI service, then invoke a function that requires a flagset", t, func() {
		grp := service.NewGroup("cli", nil)
		grp.AddService(NewCli, nil)
		grp.Invoke(func(fs *flag.FlagSet) {
			Convey("we should be able to add a flag to the flagset and parse it in Configure", func() {
				So(fs, ShouldNotBeNil)
				fs.String("foo", "baz", "usage")
				grp.Configure()
				So(fs.Parsed(), ShouldBeTrue)
				fooFlag := fs.Lookup("foo")
				So(fooFlag.Value.String(), ShouldEqual, "bar")
			})
		})
	})
	os.Args = oldArgs
}
