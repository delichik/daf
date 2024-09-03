package config

import (
	"context"
	"os"
	"testing"
	"time"
)

type testConfig struct {
	V string `yaml:"v"`
}

func (c *testConfig) Check() error {
	return nil
}

func (c *testConfig) Clone() ModuleConfig {
	return &testConfig{V: c.V}
}

func (c *testConfig) Compare(mc ModuleConfig) bool {
	ntc, ok := mc.(*testConfig)
	if !ok {
		panic("?")
	}
	return ntc.V == c.V
}

func init() {
	RegisterModuleConfig("test", &testConfig{V: "123"})
}

func TestManager_Init(t *testing.T) {
	_ = os.Remove("./test.yaml")
	defer func() {
		_ = os.Remove("./test.yaml")
	}()
	m := NewManager(context.Background(), "./test.yaml", false)
	err := m.Init()
	if err != nil {
		t.Error("init failed", err.Error())
		t.FailNow()
	}
}

func TestManager_GetModuleConfig(t *testing.T) {
	_ = os.Remove("./test.yaml")
	defer func() {
		_ = os.Remove("./test.yaml")
	}()
	m := NewManager(context.Background(), "./test.yaml", false)
	err := m.Init()
	if err != nil {
		t.Error("init failed", err.Error())
		t.FailNow()
	}
	c := m.GetModuleConfig("test")
	if c == nil {
		t.Error("get module config failed")
		t.FailNow()
	}

	_, ok := c.(*testConfig)
	if !ok {
		t.Error("convert failed")
		t.FailNow()
	}
}

func TestManager_ModifyModuleConfig(t *testing.T) {
	_ = os.Remove("./test.yaml")
	defer func() {
		_ = os.Remove("./test.yaml")
	}()
	modified := false
	m := NewManager(context.Background(), "./test.yaml", true)
	m.SetReloadCallback(func(name string, config ModuleConfig) {
		if name == "test" {
			modified = true
		}
	})
	err := m.Init()
	if err != nil {
		t.Error("init failed", err.Error())
		t.FailNow()
	}
	c := m.GetModuleConfig("test")
	if c == nil {
		t.Error("get module config failed")
		t.FailNow()
	}

	err = m.ModifyModuleConfig("test", func(config ModuleConfig) {
		lc, ok := c.(*testConfig)
		if !ok {
			t.Error("convert failed")
			t.FailNow()
		}
		lc.V = "111"
	})
	if err != nil {
		t.Error("modify module config failed", err.Error())
		t.FailNow()
	}
	time.Sleep(2 * time.Second)
	if !modified {
		t.Error("modify module config not callback")
		t.FailNow()
	}
}
