package config

import (
	"os"
	"testing"
)

func TestConfig_Load(t *testing.T) {
	_ = os.Remove("./config.yaml")
	defer func() {
		_ = os.Remove("./config.yaml")
	}()
	cfg, err := Load("./config.yaml")
	if err != nil {
		t.FailNow()
	}
	t.Log(cfg)
	cfg2, err := Load("./config.yaml")
	if err != nil {
		t.FailNow()
	}
	t.Log(cfg2)

}
