package config

import (
	"errors"
	"fmt"
	"os"

	ghodssyaml "github.com/ghodss/yaml"
	"gopkg.in/yaml.v3"

	myyaml "github.com/delichik/daf/yaml"
)

func load(path string) (*Config, error) {
	cfg := &Config{
		moduleConfigs: map[string]ModuleConfig{},
	}
	for moduleName, moduleConfig := range moduleDefaultConfigs {
		cfg.moduleConfigs[moduleName] = moduleConfig.Clone()
	}

	b, err := os.ReadFile(path)
	if err == nil {
		t := map[string]yaml.Node{}
		err = yaml.Unmarshal(b, &t)
		if err != nil {
			return nil, err
		}

		for k, v := range t {
			c, ok := cfg.moduleConfigs[k]
			if !ok {
				continue
			}
			_ = v.Decode(c)
		}
	}
	return cfg, nil
}

func save(path string, config *Config) error {
	b, err := myyaml.MarshallWithComments(config.moduleConfigs)
	if err != nil {
		return err
	}
	_ = os.Rename(path, path+".bak")
	err = os.WriteFile(path, b, 0655)
	if err != nil {
		return err
	}
	return nil
}

func check(cfg *Config) error {
	for moduleName, moduleConfig := range cfg.moduleConfigs {
		err := moduleConfig.Check()
		if err != nil {
			return fmt.Errorf("%s: %w", moduleName, err)
		}
	}
	return nil
}

func ToJson(cfg ModuleConfig) ([]byte, error) {
	yamlContent, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return ghodssyaml.YAMLToJSON(yamlContent)
}

func FromJson(moduleName string, b []byte) (ModuleConfig, error) {
	mdc, ok := moduleDefaultConfigs[moduleName]
	if !ok {
		return nil, errors.New("module not found")
	}
	mdc = mdc.Clone()

	yamlContent, err := ghodssyaml.JSONToYAML(b)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yamlContent, mdc)
	return mdc, err
}

func GetConfig[T ModuleConfig](cfg ModuleConfig) T {
	moduleConfig, ok := cfg.(T)
	if !ok {
		panic("invalid config")
	}

	return moduleConfig
}
