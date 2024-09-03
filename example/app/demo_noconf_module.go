package main

import (
	"github.com/delichik/daf"
)

type DemoNoConfModule struct {
	daf.InitializerModule
	daf.DefaultLoggerModule
}

func (m *DemoNoConfModule) ApplyConfig(_ daf.NoConfig) error {
	return nil
}

func (m *DemoNoConfModule) Name() string {
	return "demo_noconf_module"
}
