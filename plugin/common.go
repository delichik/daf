package plugin

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap/zapcore"
)

const (
	_ORDER_PREFIX = "\n\n__mfk_plugin_order__"
	_SPLITER      = "|"
	_CALL_LOGGER  = "logger"
	_CALL_REPLY   = "_reply"
)

var safeCallCache sync.Map
var locker sync.Mutex

func LogDriver() zapcore.WriteSyncer {
	return &logWriter{writer: os.Stdout}
}

type logWriter struct {
	writer io.Writer
}

func (w *logWriter) Write(data []byte) (n int, err error) {
	req := &sendObject{
		id:      0,
		call:    _CALL_LOGGER,
		content: data,
	}
	err = send(w.writer, req)
	return len(data), err
}

func (w *logWriter) Sync() error {
	return nil
}

type sendObject struct {
	id      uint64
	call    string
	content []byte
	err     error
}

func send(writer io.Writer, r *sendObject) error {
	safeData := base64.StdEncoding.EncodeToString(r.content)
	safeCall := ""
	t, loaded := safeCallCache.Load(r.call)
	if loaded {
		safeCall = t.(string)
	} else {
		safeCall = base64.StdEncoding.EncodeToString([]byte(r.call))
		safeCallCache.Store(r.call, safeCall)
	}

	locker.Lock()
	defer locker.Unlock()

	if _, err := fmt.Fprintln(writer, _ORDER_PREFIX); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, strconv.FormatUint(r.id, 10)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, _SPLITER); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, safeCall); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, _SPLITER); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(writer, safeData); err != nil {
		return err
	}
	return nil
}

func read(reader *bufio.Reader) (*sendObject, error) {
	line := ""
	for {
		p, prefix, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}
		line += string(p)
		if prefix {
			continue
		}
		t := line
		line = ""
		if strings.HasPrefix(t, _ORDER_PREFIX) {
			parts := strings.Split(t[len(_ORDER_PREFIX):], _SPLITER)
			if len(parts) != 3 {
				continue
			}

			rsp := &sendObject{}
			rsp.id, err = strconv.ParseUint(parts[0], 10, 0)
			if err != nil {
				rsp.err = fmt.Errorf("decode id failed: %w", err)
				return rsp, nil
			}

			call, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				rsp.err = fmt.Errorf("decode call failed: %w", err)
				return rsp, nil
			}
			rsp.call = string(call)
			rsp.content, err = base64.StdEncoding.DecodeString(parts[2])
			if err != nil {
				rsp.err = fmt.Errorf("decode content failed: %w", err)
				return rsp, nil
			}
			return rsp, nil
		}
	}
}

type HandshakeInfo struct {
	Name    string
	Version string
}

func checkHandshake(handshake string, options *Options) bool {
	data, err := base64.StdEncoding.DecodeString(handshake)
	if err != nil {
		return false
	}
	info := &HandshakeInfo{}
	err = msgpack.Unmarshal(data, info)
	if err != nil {
		return false
	}
	if info.Name != options.HostName {
		return false
	}

	if info.Version < options.HostMinimalVersion {
		return false
	}

	return true
}
