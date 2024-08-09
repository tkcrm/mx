package connectrpc_transport

// Config provides configuration for grpc server.
type Config struct {
	Enabled        bool   `default:"true" usage:"allows to enable server" example:"true"`
	Addr           string `default:":9000" validate:"required,hostname_port" usage:"server listen address" example:"localhost:9000"`
	ReflectEnabled bool   `yaml:"reflect_enabled" default:"false" usage:"allows to enable reflection service" example:"false"`
}
