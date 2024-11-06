package daf

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"reflect"
	"runtime"
	"syscall"

	"go.uber.org/zap"

	"github.com/delichik/daf/config"
	"github.com/delichik/daf/logger"
	"github.com/delichik/daf/utils"
)

var (
	gCtx          context.Context
	gCancel       context.CancelFunc
	beforeRunCall func()
	afterRunCall  func()

	cm                   *config.Manager
	disableRefreshConfig bool

	modules             map[string]*ModuleEntry
	orderedModules      []*ModuleEntry
	autoLoadModuleCount int

	flagSet *flag.FlagSet
)

func init() {
	flagSet = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	gCtx, gCancel = context.WithCancel(context.Background())
	modules = map[string]*ModuleEntry{}
}

type AllowedFlagProtoVar interface {
	int | string | bool | float64
}

type internalCommandLineVars struct {
	ConfigPath string
	Help       bool
	Version    bool
}

func RegisterFlagVar[T AllowedFlagProtoVar](value *T, name string, defaultValue T, usage string) {
	switch v := any(value).(type) {
	case *int:
		dv := any(defaultValue).(int)
		flagSet.IntVar(v, name, dv, usage)
	case *string:
		dv := any(defaultValue).(string)
		flagSet.StringVar(v, name, dv, usage)
	case *bool:
		dv := any(defaultValue).(bool)
		flagSet.BoolVar(v, name, dv, usage)
	case *float64:
		dv := any(defaultValue).(float64)
		flagSet.Float64Var(v, name, dv, usage)
	default:
		panic("unsupported flag type: " + reflect.TypeOf(value).String())
	}
}

// Deprecated: Now framework will detect auto-loading automatically.
// Use RegisterModule instead.
func RegisterAutoLoadModule[T config.ModuleConfig](module Module[T]) {
	if utils.IsCalledByMainDirect(1) {

	} else if utils.IsCalledByInit(1) {
		autoLoadModuleCount++
	} else {
		panic("module must be registered in init() or directly under main()")
	}
	registerModule(module)
}

// RegisterModule Register a module.
// The module registered in main() directly, be called as invoked module, will
// always be loaded.
// The module registered in init(), be called as auto module, will be loaded when
// it has been used in invoked module.
func RegisterModule[T config.ModuleConfig](module Module[T]) {
	if utils.IsCalledByMainDirect(1) {
		// just pass-through
	} else if utils.IsCalledByInit(1) {
		autoLoadModuleCount++
	} else {
		panic("module must be registered in init() or directly under main()")
	}
	registerModule(module)
}

// DisableRefreshConfig Disable auto refresh config
func DisableRefreshConfig() {
	disableRefreshConfig = true
}

// BeforeRun To execute codes before Run().
func BeforeRun(call func()) {
	beforeRunCall = call
}

// AfterRun To execute codes after Run().
func AfterRun(call func()) {
	afterRunCall = call
}

// Run Prepare the configs and start up the app.
func Run(version string) {
	ctx, cancel := context.WithCancel(gCtx)
	clvs := parseFlags(version)
	cm = config.NewManager(ctx, clvs.ConfigPath, !disableRefreshConfig)
	for _, module := range orderedModules {
		module.SetConfigManager(cm)
		if module.noConfig {
			continue
		}
		if module.AdditionalLogger() {
			logger.RegisterAdditionalLogger(module.Name())
		}
	}

	err := cm.Init()
	if err != nil {
		log.Printf("Init config failed: %s, exit", err.Error())
		cancel()
		return
	}
	cm.SetReloadCallback(ReloadConfig)
	logger.InitDefault(cm)
	for _, module := range orderedModules {
		if module.AdditionalLogger() {
			logger.Init(module.Name(), cm)
		}
	}
	logger.Info("App init", zap.String("version", version))

	if beforeRunCall != nil {
		beforeRunCall()
	}
	logger.Debug("Loading app modules")
	for i, module := range orderedModules {
		err = module.OnInit(ctx)
		if err != nil {
			logger.Fatal("Init module failed",
				zap.String("name", module.Name()),
				zap.Error(err))
		}
		logger.Debug("Prepare module",
			zap.String("name", module.Name()),
			zap.Bool("auto_loaded", i < autoLoadModuleCount))
		if !module.noConfig {
			cfg := cm.GetModuleConfig(module.Name())
			if cfg == nil {
				logger.Warn("Module has no config, but requires a config", zap.String("name", module.Name()))
				continue
			}
			logger.Debug("Applying module config", zap.String("name", module.Name()))
			err := module.ApplyConfig(cfg)
			if err != nil {
				logger.Fatal("Apply module config failed, exit",
					zap.String("name", module.Name()),
					zap.Error(err))
			}
		}
		err = module.OnRun()
		if err != nil {
			logger.Fatal("Run module failed",
				zap.String("name", module.Name()),
				zap.Error(err))
		}
	}
	logger.Debug("App modules loaded")
	if afterRunCall != nil {
		afterRunCall()
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGABRT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case <-signalChan:
	case <-gCtx.Done():
	}
	signal.Stop(signalChan)
	logger.Warn("App shutdown")

	cancel()
	for _, module := range orderedModules[autoLoadModuleCount:] {
		module.OnExit()
	}
	for _, module := range orderedModules[:autoLoadModuleCount] {
		module.OnExit()
	}
}

// Shutdown Stop the app and exit.
func Shutdown() {
	gCancel()
}

func parseFlags(version string) internalCommandLineVars {
	clvs := internalCommandLineVars{}
	flagSet.StringVar(&clvs.ConfigPath, "config", "config.yaml", "Config path")
	flagSet.BoolVar(&clvs.Help, "help", false, "Print help")
	flagSet.BoolVar(&clvs.Version, "version", false, "Print version")
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		panic("parse flags failed: " + err.Error())
	}
	if clvs.Help {
		flagSet.Usage()
		os.Exit(1)
	}
	if clvs.Version {
		fmt.Println("Go version:\t\t", runtime.Version())
		fmt.Println("Binary version:\t", version)
		os.Exit(1)
	}
	return clvs
}

func registerModule[T config.ModuleConfig](module Module[T]) {
	existedModule, ok := modules[module.Name()]
	if ok {
		panic(fmt.Errorf("module %s already registered by %s",
			existedModule.Name(), existedModule.registerer))
	}

	moduleEntry := newModuleEntry(module)
	modules[module.Name()] = moduleEntry
	orderedModules = append(orderedModules, moduleEntry)

	rt := reflect.TypeFor[T]()
	rv := reflect.Zero(rt)
	if rt.Kind() == reflect.Pointer {
		rv = reflect.New(rt.Elem())
	}

	if !rt.Implements(noConfigIfaceType) {
		cfg := rv.Interface().(config.ModuleConfig)
		config.RegisterModuleConfig(module.Name(), cfg)
	} else {
		moduleEntry.noConfig = true
	}
}

func ReloadConfig(name string, cfg config.ModuleConfig) {
	module, ok := modules[name]
	if !ok {
		return
	}
	logger.Info("Reloading module config", zap.String("name", name))
	err := module.ApplyConfig(cfg)
	if err != nil {
		var fatalError *FatalError
		ok = errors.As(err, &fatalError)
		if ok {
			logger.Fatal("Apply module config failed with fatal error",
				zap.String("name", name),
				zap.Error(fatalError))
		} else {
			logger.Error("Apply module config failed",
				zap.String("name", name),
				zap.Error(err))
		}
	}
}
