package config

import (
	"fmt"

	myyaml "github.com/delichik/daf/yaml"
)

type ConfigSet interface {
	// GetModuleConfig 获取 Config。
	GetModuleConfig(moduleName string) ModuleConfig
}

type ModuleConfig interface {
	// Check 预检查错误
	Check() error
	// Clone 深度拷贝内容并返回
	Clone() ModuleConfig
	// Compare 对比两个同类型配置是否一样, 一样则返回 true
	Compare(ModuleConfig) bool
}

var moduleDefaultConfigs = map[string]ModuleConfig{}

type Config struct {
	moduleConfigs map[string]ModuleConfig
}

func (c *Config) GetModuleConfig(moduleName string) ModuleConfig {
	return c.moduleConfigs[moduleName]
}

func (c *Config) String() string {
	b, err := myyaml.MarshallWithComments(c.moduleConfigs)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func RegisterModuleConfig(name string, defaultConfig ModuleConfig) {
	_, ok := moduleDefaultConfigs[name]
	if ok {
		panic(fmt.Errorf("module name %s existed", name))
	}

	moduleDefaultConfigs[name] = defaultConfig
}

func Load(path string) (*Config, error) {
	cfg, err := load(path)
	if err != nil {
		return nil, err
	}

	err = save(path, cfg)
	if err != nil {
		return nil, err
	}

	err = check(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
