package connectrpc_client

type logger interface {
	Info(...any)
	Infof(template string, args ...any)
}
