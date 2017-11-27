package component

import (
	"testing"
	"time"

	"github.com/anuvu/zlog"
	. "github.com/smartystreets/goconvey/convey"
)

func TestContext(t *testing.T) {
	Convey("After we create a context", t, func() {
		ctx := RootContext(zlog.New("test")).(*srvCtx)
		go func() {
			<-ctx.Ctx().Done()
			ctx.Log().Info().Msg("Done")
		}()
		time.Sleep(time.Second)
		ctx.Shutdown()
	})
}
