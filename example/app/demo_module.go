package main

import (
	"context"
	"strconv"
	"time"

	"github.com/delichik/go-pkgs/deep"

	"github.com/delichik/daf"
	"github.com/delichik/daf/config"
	"github.com/delichik/daf/logger"
	"github.com/delichik/daf/utils"
)

type DemoModuleConfig = *demoModuleConfig

type demoModuleConfig struct {
	ListenAddr string `yaml:"listen-addr" comment:"Address to listen, 0.0.0.0 for default"`
	ListenPort int    `yaml:"listen-port" comment:"Port to listen, 80 for default"`
}

func (c *demoModuleConfig) Check() error {
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0"
	}

	if c.ListenPort <= 0 || c.ListenPort > 65534 {
		c.ListenPort = 80
	}
	return nil
}

func (c *demoModuleConfig) Clone() config.ModuleConfig {
	n := &demoModuleConfig{}
	deep.Copy(c, n)
	return n
}

func (c *demoModuleConfig) Compare(moduleConfig config.ModuleConfig) bool {
	return utils.DeepCompare(c, moduleConfig)
}

type DemoModule struct {
	daf.DefaultLoggerModule
}

func (m *DemoModule) Name() string {
	return "demo_module"
}

func (m *DemoModule) ApplyConfig(_ *demoModuleConfig) error {
	logger.Info("Apply config")
	return nil
}

func (m *DemoModule) OnInit(_ context.Context) error {
	return nil
}

func (m *DemoModule) OnRun() error {
	logger.Info("On run")
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for i := 5; i > 0; i-- {
			logger.Info("Shutdown after " + strconv.Itoa(i))
			<-ticker.C
		}
		daf.Shutdown()
	}()
	return nil
}

func (m *DemoModule) OnExit() {
	logger.Info("On exit")
}
