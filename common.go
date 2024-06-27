package daf

import (
	"context"
	"reflect"

	"github.com/delichik/daf/config"
)

var noConfigIfaceType = reflect.TypeOf((*noConfigIface)(nil)).Elem()

type noConfigIface interface {
	__NoConfig()
}
type NoConfig = *noConfig

type noConfig struct{}

func (c *noConfig) Check() error {
	return nil
}

func (c *noConfig) Clone() config.ModuleConfig {
	return c
}

func (c *noConfig) Compare(_ config.ModuleConfig) bool {
	return true
}

func (c *noConfig) __NoConfig() {}

type AdditionalLoggerModule struct{}

func (m *AdditionalLoggerModule) SetConfigManager(_ *config.Manager) {}

func (m *AdditionalLoggerModule) AdditionalLogger() bool {
	return true
}

type DefaultLoggerModule struct{}

func (m *DefaultLoggerModule) SetConfigManager(_ *config.Manager) {}

func (m *DefaultLoggerModule) AdditionalLogger() bool {
	return false
}

type InitializerModule struct{}

func (m *InitializerModule) OnInit(_ context.Context) error {
	return nil
}

func (m *InitializerModule) OnRun() error {
	return nil
}

func (m *InitializerModule) OnExit() {}
