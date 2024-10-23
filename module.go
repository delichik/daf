package daf

import (
	"context"
	"reflect"
	"runtime"
	"strconv"

	"github.com/delichik/daf/config"
)

type Module[T config.ModuleConfig] interface {
	// Name 返回模块的名字，所有模块名字不能重复
	Name() string

	// ApplyConfig 触发配置应用，当启动和配置发生变化时会被调用
	ApplyConfig(cfg T) error

	// OnInit 初始化模块，当应用初始化时会被调用
	OnInit(ctx context.Context) error

	// OnRun 启动模块，当应用启动时会被调用
	OnRun() error

	// AdditionalLogger 返回是否需要额外的日志记录
	AdditionalLogger() bool

	// OnExit 触发模块退出，当应用退出时会被调用
	OnExit()

	SetConfigManager(cm *config.Manager)
}

type ModuleEntry struct {
	module     any
	registerer string
	noConfig   bool

	_funName             reflect.Value
	_funApplyConfig      reflect.Value
	_funOnInit           reflect.Value
	_funOnRun            reflect.Value
	_funAdditionalLogger reflect.Value
	_funOnExit           reflect.Value
	_funSetConfigManager reflect.Value
}

func nullableError(in any) error {
	switch r := in.(type) {
	case error:
		return r
	default:
		return nil
	}
}

func mustGetFunc(v reflect.Value, name string) reflect.Value {
	f := v.MethodByName(name)
	if f.IsNil() || !f.IsValid() || f.IsZero() {
		panic("method not found: " + name)
	}
	return f
}

func newModuleEntry(module any) *ModuleEntry {
	_, file, line, _ := runtime.Caller(3)
	moduleEntry := &ModuleEntry{
		module:     module,
		registerer: file + ":" + strconv.Itoa(line),
	}

	rv := reflect.ValueOf(module)
	moduleEntry._funName = mustGetFunc(rv, "Name")
	moduleEntry._funApplyConfig = mustGetFunc(rv, "ApplyConfig")
	moduleEntry._funOnInit = mustGetFunc(rv, "OnInit")
	moduleEntry._funOnRun = mustGetFunc(rv, "OnRun")
	moduleEntry._funAdditionalLogger = mustGetFunc(rv, "AdditionalLogger")
	moduleEntry._funOnExit = mustGetFunc(rv, "OnExit")
	moduleEntry._funSetConfigManager = mustGetFunc(rv, "SetConfigManager")
	return moduleEntry
}

func (m *ModuleEntry) Name() string {
	res := m._funName.Call([]reflect.Value{})
	return (res[0].Interface()).(string)
}

func (m *ModuleEntry) ApplyConfig(cfg config.ModuleConfig) error {
	res := m._funApplyConfig.Call([]reflect.Value{reflect.ValueOf(cfg)})
	return nullableError(res[0].Interface())
}

func (m *ModuleEntry) OnInit(ctx context.Context) error {
	res := m._funOnInit.Call([]reflect.Value{reflect.ValueOf(ctx)})
	return nullableError(res[0].Interface())
}

func (m *ModuleEntry) OnRun() error {
	res := m._funOnRun.Call([]reflect.Value{})
	return nullableError(res[0].Interface())
}

func (m *ModuleEntry) AdditionalLogger() bool {
	res := m._funAdditionalLogger.Call([]reflect.Value{})
	return (res[0].Interface()).(bool)
}

func (m *ModuleEntry) OnExit() {
	m._funOnExit.Call([]reflect.Value{})
}

func (m *ModuleEntry) SetConfigManager(cm *config.Manager) {
	m._funSetConfigManager.Call([]reflect.Value{reflect.ValueOf(cm)})
}
