package plugin

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"

	"github.com/delichik/mfk/logger"
)

type Host struct {
	plugins map[string]*Entity
	name    string
	version string
	e       Executor
}

func NewHost(name string, version string, e Executor) *Host {
	return &Host{
		plugins: make(map[string]*Entity),
		name:    name,
		version: version,
		e:       e,
	}
}

func (h *Host) Load(pluginPath string) error {
	handshake, err := msgpack.Marshal(&HandshakeInfo{
		Name:    h.name,
		Version: h.version,
	})
	if err != nil {
		panic(err)
	}

	handshakeStr := base64.StdEncoding.EncodeToString(handshake)

	entries, err := os.ReadDir(pluginPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".exe")
		logger.Info("load plugin", zap.String("name", name))
		cmd := exec.Command(pluginPath+"/"+entry.Name(), "-h", handshakeStr)
		h.plugins[name] = newEntity(name, cmd, h)
		err = h.plugins[name].Start()
		if err != nil {
			logger.Error("fail to load plugin", zap.String("name", entry.Name()), zap.Error(err))
		}
	}

	return nil
}

func (h *Host) Call(call string, data []byte) ([]byte, error) {
	for _, plg := range h.plugins {
		return plg.CallWithResponse(call, data)
	}
	return []byte(""), nil
}

func (h *Host) Notice(call string, data []byte) ([]byte, error) {
	for _, plg := range h.plugins {
		plg.Call(call, data)
		return []byte(""), nil
	}
	return []byte(""), nil
}

func (h *Host) dispatchCall(e *Entity, call string, data []byte, replyFunc func([]byte) error) {
	switch call {
	case _CALL_LOGGER:
		log(e.name, data)
	default:
		reply, err := h.e.OnCall(call, data)
		if err != nil {
			replyFunc([]byte(err.Error()))
			return
		}
		replyFunc(reply)
	}
}

type logContent struct {
	Level   string `json:"level"`
	Caller  string `json:"caller"`
	Message string `json:"msg"`
}

func log(pluginName string, data []byte) {
	c := logContent{}
	err := json.Unmarshal(data, &c)
	if err != nil {
		return
	}

	fieldMap := map[string]interface{}{}
	_ = json.Unmarshal(data, &fieldMap)
	fields := []zap.Field{}
	fields = append(fields, zap.String("plugin_name", pluginName), zap.String("plugin_caller", c.Caller))
	for k, v := range fieldMap {
		if k == "level" ||
			k == "ts" ||
			k == "caller" ||
			k == "msg" ||
			k == "plugin_name" ||
			k == "plugin_caller" {
			continue
		}
		fields = append(fields, zap.Any(k, v))
	}
	c.Message = "[plugin] " + c.Message
	switch c.Level {
	case "debug":
		logger.Debug(c.Message, fields...)
	case "info":
		logger.Info(c.Message, fields...)
	case "warn":
		logger.Warn(c.Message, fields...)
	case "error":
		logger.Error(c.Message, fields...)
	}
}
