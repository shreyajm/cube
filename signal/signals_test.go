package signal

import (
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/anuvu/cube/service"
	. "github.com/smartystreets/goconvey/convey"
)

type sigH struct {
	s []os.Signal
	l *sync.RWMutex
}

func (s *sigH) handle(sig os.Signal) {
	s.l.Lock()
	s.s = append(s.s, sig)
	s.l.Unlock()
}

func (s *sigH) Len() int {
	s.l.RLock()
	defer s.l.RUnlock()
	return len(s.s)
}

func (s *sigH) Sig(i int) os.Signal {
	return s.s[i]
}

func TestSignals(t *testing.T) {
	Convey("Create a signal Router", t, func() {
		ctx := service.NewContext()
		s := NewSignalRouter()
		So(s, ShouldNotBeNil)
		Convey("Should be able add handler", func() {
			So(s.IsIgnored(syscall.SIGINT), ShouldBeFalse)
			So(s.IsHandled(syscall.SIGINT), ShouldBeFalse)
			s.Handle(syscall.SIGINT, func(os.Signal) {})
			So(s.IsHandled(syscall.SIGINT), ShouldBeTrue)
		})
		Convey("Should be able to reset handler", func() {
			s.Reset(syscall.SIGINT)
			So(s.IsHandled(syscall.SIGINT), ShouldBeFalse)
		})
		Convey("Should be able to ignore signal", func() {
			s.Ignore(syscall.SIGINT)
			So(s.IsIgnored(syscall.SIGINT), ShouldBeTrue)
			So(s.IsHandled(syscall.SIGINT), ShouldBeFalse)
		})

		Convey("Should be able to start the service", func() {
			sh := &sigH{[]os.Signal{}, &sync.RWMutex{}}
			s.Handle(syscall.SIGINT, sh.handle)
			So(s.IsHandled(syscall.SIGINT), ShouldBeTrue)
			So(len(sh.s), ShouldEqual, 0)
			// Check lifecycle
			So(s.(service.HealthHook).IsHealthy(ctx), ShouldBeFalse)
			s.(service.StartHook).Start(ctx)
			So(s.(service.HealthHook).IsHealthy(ctx), ShouldBeTrue)

			Convey("Should be able to handle a signal", func() {
				// Fire a signal
				s.(*router).signalCh <- syscall.SIGINT

				// Sleep for a second
				time.Sleep(time.Second)
				So(sh.Len(), ShouldEqual, 1)
				So(sh.Sig(0), ShouldEqual, syscall.SIGINT)
			})

			Convey("Should be able to stop the service", func() {
				So(s.(service.HealthHook).IsHealthy(ctx), ShouldBeTrue)
				ctx.Shutdown()
				time.Sleep(time.Second)
				So(s.(service.HealthHook).IsHealthy(ctx), ShouldBeFalse)
			})
		})
	})
}
