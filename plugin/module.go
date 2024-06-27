package plugin

type Handler func(name string, data []byte) ([]byte, error)

type Plugin interface {
	Init() error
	UnInit()
	Handle(data []byte) ([]byte, error)
}

type Executor interface {
	OnCall(call string, data []byte) ([]byte, error)
}
