package main

import (
	"github.com/delichik/daf/app"
)

type DemoNoConfModule struct {
	app.InitializerModule
	app.DefaultLoggerModule
}

func (m *DemoNoConfModule) ApplyConfig(_ app.NoConfig) error {
	return nil
}

func (m *DemoNoConfModule) Name() string {
	return "demo_noconf_module"
}
