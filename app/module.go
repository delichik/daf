package app

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

	Name             func() string
	ApplyConfig      func(cfg config.ModuleConfig) error
	OnInit           func(ctx context.Context) error
	OnRun            func() error
	AdditionalLogger func() bool
	OnExit           func()
	SetConfigManager func(cm *config.Manager)
}

func nullableError(in any) error {
	switch r := in.(type) {
	case error:
		return r
	default:
		return nil
	}
}

func newModuleEntry(module any) *ModuleEntry {
	_, file, line, _ := runtime.Caller(3)
	moduleEntry := &ModuleEntry{
		module:     module,
		registerer: file + ":" + strconv.Itoa(line),
	}

	rv := reflect.ValueOf(module)
	moduleEntry._funName = rv.MethodByName("Name")
	moduleEntry.Name = func() string {
		res := moduleEntry._funName.Call([]reflect.Value{})
		return (res[0].Interface()).(string)
	}

	moduleEntry._funApplyConfig = rv.MethodByName("ApplyConfig")
	moduleEntry.ApplyConfig = func(cfg config.ModuleConfig) error {
		res := moduleEntry._funApplyConfig.Call([]reflect.Value{reflect.ValueOf(cfg)})
		return nullableError(res[0].Interface())
	}

	moduleEntry._funOnInit = rv.MethodByName("OnInit")
	moduleEntry.OnInit = func(ctx context.Context) error {
		res := moduleEntry._funOnInit.Call([]reflect.Value{reflect.ValueOf(ctx)})
		return nullableError(res[0].Interface())
	}

	moduleEntry._funOnRun = rv.MethodByName("OnRun")
	moduleEntry.OnRun = func() error {
		res := moduleEntry._funOnRun.Call([]reflect.Value{})
		return nullableError(res[0].Interface())
	}

	moduleEntry._funAdditionalLogger = rv.MethodByName("AdditionalLogger")
	moduleEntry.AdditionalLogger = func() bool {
		res := moduleEntry._funAdditionalLogger.Call([]reflect.Value{})
		return (res[0].Interface()).(bool)
	}

	moduleEntry._funOnExit = rv.MethodByName("OnExit")
	moduleEntry.OnExit = func() {
		moduleEntry._funOnExit.Call([]reflect.Value{})
	}

	moduleEntry._funSetConfigManager = rv.MethodByName("SetConfigManager")
	moduleEntry.SetConfigManager = func(cm *config.Manager) {
		moduleEntry._funSetConfigManager.Call([]reflect.Value{reflect.ValueOf(cm)})
	}

	return moduleEntry
}
