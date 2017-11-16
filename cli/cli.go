package cli

import (
	"flag"
	"os"

	"github.com/anuvu/cube/service"
)

// NewCli is the CLI service constructor
func NewCli(ctx service.Context) *flag.FlagSet {
	flagSet := flag.NewFlagSet("cube", flag.ExitOnError)

	ctx.AddLifecycle(&service.Lifecycle{
		ConfigHook: func() {
			flagSet.Parse(os.Args[1:])
		},
	})

	return flagSet
}
