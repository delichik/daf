package plugin

import (
	"bufio"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type callRequest struct {
	channel chan *callResponse
	addTime int64
}
type callResponse struct {
	err     error
	content []byte
}

type Entity struct {
	name string
	cmd  *exec.Cmd
	host *Host

	ctx    context.Context
	cancel context.CancelFunc

	pluginOutput   io.ReadCloser
	pluginInput    io.WriteCloser
	stdoutBuffered *bufio.Reader

	calls       map[uint64]*callRequest
	callLocker  sync.RWMutex
	callIDIndex atomic.Uint64
}

func newEntity(name string, cmd *exec.Cmd, host *Host) *Entity {
	ctx, cancel := context.WithCancel(context.Background())
	e := &Entity{
		name:   name,
		cmd:    cmd,
		host:   host,
		ctx:    ctx,
		cancel: cancel,
		calls:  map[uint64]*callRequest{},
	}
	return e
}

func (e *Entity) Start() error {
	var err error
	e.pluginOutput, err = e.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	e.pluginInput, err = e.cmd.StdinPipe()
	if err != nil {
		return err
	}
	e.stdoutBuffered = bufio.NewReader(e.pluginOutput)

	err = e.cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		for {
			rsp, err := read(e.stdoutBuffered)
			if err != nil || e.ctx.Err() != nil {
				return
			}

			if strings.HasSuffix(rsp.call, _CALL_REPLY) {
				e.callLocker.Lock()
				req, ok := e.calls[rsp.id]
				if ok {
					delete(e.calls, rsp.id)
					select {
					case req.channel <- &callResponse{
						err:     rsp.err,
						content: rsp.content,
					}:
					case <-e.ctx.Done():
						return
					default:
					}
					close(req.channel)
				}
				e.callLocker.Unlock()
				continue
			}
			e.host.dispatchCall(e, rsp.call, rsp.content, e.newReplyFunc(rsp.id, rsp.call))
		}
	}()

	go func() {
		timer := time.NewTimer(500 * time.Millisecond)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				e.callLocker.Lock()
				for id, req := range e.calls {
					if time.Now().Unix()-req.addTime > 5 {
						close(req.channel)
						delete(e.calls, id)
					}
				}
				e.callLocker.Unlock()
				timer.Reset(500 * time.Millisecond)
			case <-e.ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (e *Entity) newReplyFunc(id uint64, cmd string) func(data []byte) error {
	return func(data []byte) error {
		return send(e.pluginInput, &sendObject{
			id:      id,
			call:    cmd + _CALL_REPLY,
			content: data,
		})
	}
}

func (e *Entity) Stop() error {
	e.cancel()
	return e.cmd.Process.Kill()
}

func (e *Entity) Call(call string, data []byte) error {
	req := &sendObject{
		id:      e.callIDIndex.Add(1),
		call:    call,
		content: data,
	}
	return send(e.pluginInput, req)
}

func (e *Entity) CallWithResponse(call string, data []byte) ([]byte, error) {
	req := &sendObject{
		id:      e.callIDIndex.Add(1),
		call:    call,
		content: data,
	}
	channel := make(chan *callResponse, 1)
	e.callLocker.Lock()
	e.calls[req.id] = &callRequest{
		channel: channel,
		addTime: time.Now().Unix(),
	}
	e.callLocker.Unlock()
	err := send(e.pluginInput, req)
	if err != nil {
		return nil, err
	}
	rsp := <-channel
	e.callLocker.Lock()
	o, ok := e.calls[req.id]
	if ok {
		close(o.channel)
		delete(e.calls, req.id)
	}
	e.callLocker.Unlock()
	return rsp.content, rsp.err
}
