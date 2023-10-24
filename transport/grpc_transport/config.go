package grpc_transport

// Config provides configuration for grpc server.
type Config struct {
	Enabled bool   `default:"true" usage:"allows to enable grpc server" example:"true"`
	Reflect bool   `default:"false" usage:"allows to enable grpc reflection service" example:"false"`
	Addr    string `default:":9000" validate:"required,hostname_port" usage:"grpc server listen address" example:"localhost:9000"`
	Network string `default:"tcp" required:"true" validate:"oneof=tcp udp" usage:"grpc server listen network: tpc/udp" example:"tcp"`
}
