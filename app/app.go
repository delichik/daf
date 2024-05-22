package app

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
)

var (
	gCtx                 context.Context
	gCancel              context.CancelFunc
	cm                   *config.Manager
	beforeRunCall        func()
	afterRunCall         func()
	modules              map[string]*ModuleEntry
	orderedModules       []*ModuleEntry
	autoLoadModuleCount  int
	disableRefreshConfig bool
)

func init() {
	gCtx, gCancel = context.WithCancel(context.Background())
	modules = map[string]*ModuleEntry{}
}

type CommandLineVars struct {
	ConfigPath string
	Help       bool
	Version    bool
}

func parseFlags(version string) CommandLineVars {
	clvs := CommandLineVars{}
	flag.StringVar(&clvs.ConfigPath, "c", "config.yaml", "Config path")
	flag.BoolVar(&clvs.Help, "h", false, "Print help")
	flag.BoolVar(&clvs.Version, "v", false, "Print version")
	flag.Parse()
	if clvs.Help {
		flag.Usage()
		os.Exit(1)
	}
	if clvs.Version {
		fmt.Println("Go version:\t\t", runtime.Version())
		fmt.Println("Binary version:\t", version)
		os.Exit(1)
	}
	return clvs
}

func RegisterAutoLoadModule[T config.ModuleConfig](module Module[T]) {
	registerModule(module)
	autoLoadModuleCount++
}

func RegisterModule[T config.ModuleConfig](module Module[T]) {
	registerModule(module)
}

func DisableRefreshConfig() {
	disableRefreshConfig = true
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

	var cfg T
	rt := reflect.TypeOf(cfg)
	if !rt.Implements(noConfigIfaceType) {
		config.RegisterModuleConfig(module.Name(), cfg)
	}
}

func BeforeRun(call func()) {
	beforeRunCall = call
}

func AfterRun(call func()) {
	afterRunCall = call
}

func Run(version string) {
	ctx, cancel := context.WithCancel(context.Background())
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
	logger.Info("Loading app modules")
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
				logger.Warn("Skip module", zap.String("name", module.Name()))
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
	logger.Info("App modules loaded")
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
	logger.Info("App shutdown")

	cancel()
	for _, module := range orderedModules {
		if !module.noConfig {
			cfg := cm.GetModuleConfig(module.Name())
			if cfg == nil {
				continue
			}
		}
		module.OnExit()
	}
}

func Shutdown() {
	gCancel()
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
