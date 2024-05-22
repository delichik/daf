package config

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

type ReloadCallback func(name string, config ModuleConfig)

type Manager struct {
	path  string
	ctx   context.Context
	wg    sync.WaitGroup
	watch bool

	locker         sync.RWMutex
	loadConfigChan chan struct{}
	c              *Config
	lastLoadC      *Config
	reloadCallback ReloadCallback
}

func NewManager(ctx context.Context, path string, watch bool) *Manager {
	return &Manager{
		path:           path,
		ctx:            ctx,
		loadConfigChan: make(chan struct{}, 1),
		watch:          watch,
	}
}

func (m *Manager) Init() error {
	newConfig, err := Load(m.path)
	if err != nil {
		return err
	}

	m.locker.Lock()
	defer m.locker.Unlock()
	m.lastLoadC = newConfig
	m.c = &Config{moduleConfigs: map[string]ModuleConfig{}}
	for moduleName, moduleConfig := range newConfig.moduleConfigs {
		m.c.moduleConfigs[moduleName] = moduleConfig.Clone()
	}

	err = save(m.path, m.c)
	if err != nil {
		return err
	}

	if m.watch {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.watchConfig()
		}()
	}
	return nil
}

func (m *Manager) watchConfig() {
	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			m.compareAndUpdateConfig()
		case <-m.loadConfigChan:
			m.compareAndUpdateConfig()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) compareAndUpdateConfig() {
	m.locker.Lock()
	defer m.locker.Unlock()
	newConfig, err := load(m.path)
	if err != nil {
		log.Println("Config file load failed:", err.Error(), ", abort to load")
		return
	}
	err = check(newConfig)
	if err != nil {
		log.Println("Config file load failed:", err.Error(), ", abort to load")
		return
	}
	changed := false
	for name, lmc := range m.lastLoadC.moduleConfigs {
		nmc := newConfig.GetModuleConfig(name)
		if !nmc.Compare(lmc) {
			changed = true
			if m.reloadCallback != nil {
				m.reloadCallback(name, nmc)
			}
		}
	}
	if changed {
		m.c = &Config{moduleConfigs: map[string]ModuleConfig{}}
		m.lastLoadC = &Config{moduleConfigs: map[string]ModuleConfig{}}
		for moduleName, moduleConfig := range newConfig.moduleConfigs {
			m.c.moduleConfigs[moduleName] = moduleConfig.Clone()
			m.lastLoadC.moduleConfigs[moduleName] = moduleConfig.Clone()
		}
	}
}

func (m *Manager) GetModuleConfig(moduleName string) ModuleConfig {
	m.locker.RLock()
	defer m.locker.RUnlock()
	return m.lastLoadC.moduleConfigs[moduleName]
}

func (m *Manager) ModifyModuleConfig(moduleName string, call func(ModuleConfig)) error {
	m.locker.Lock()
	defer m.locker.Unlock()
	mc, ok := m.c.moduleConfigs[moduleName]
	if !ok {
		return errors.New("module config not found")
	}
	call(mc)
	err := save(m.path, m.c)
	if err != nil {
		return err
	}
	select {
	case m.loadConfigChan <- struct{}{}:
	default:
	}
	return nil
}

func (m *Manager) SetReloadCallback(callback ReloadCallback) {
	m.reloadCallback = callback
}
