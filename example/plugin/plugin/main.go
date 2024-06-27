package main

import (
	"github.com/delichik/mfk/logger"
	"github.com/delichik/mfk/plugin"
)

func main() {
	logger.InitDefault(&logger.Config{
		Level:     "debug",
		Format:    "json",
		LogDriver: plugin.LogDriver(),
	})
	plugin.RegisterHandler("hello", &ExamplePlugin{})

	plugin.RunPlugin(&plugin.Options{
		Name:               "example-plugin-plugin",
		Version:            "0.0.1",
		HostName:           "example-plugin-host",
		HostMinimalVersion: "0.0.1",
	})
}

type ExamplePlugin struct {
}

func (p *ExamplePlugin) Init() error {
	logger.Info("init plugin")
	return nil
}

func (p *ExamplePlugin) UnInit() {
	logger.Info("uninit plugin")
}

func (p *ExamplePlugin) Handle(data []byte) ([]byte, error) {
	return []byte("hello example-plugin-host"), nil
}
