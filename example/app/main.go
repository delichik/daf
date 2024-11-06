package main

import (
	"github.com/delichik/daf"
	"github.com/delichik/daf/logger"
)

func init() {
	daf.RegisterModule[daf.NoConfig](&DemoNoConfModule{})
}

func main() {
	daf.BeforeRun(func() {
		logger.Info("Before run")
	})
	daf.AfterRun(func() {
		logger.Info("After run")
	})
	daf.RegisterModule[DemoModuleConfig](&DemoModule{})
	daf.Run("0.0.1")
}
