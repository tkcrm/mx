package grpc_transport

// Config provides configuration for http server.
type Config struct {
	Enabled bool   `default:"true" usage:"allows to enable grpc server"`
	Reflect bool   `default:"false" usage:"allows to enable grpc reflection service"`
	Addr    string `default:":9000" validate:"hostname_port" usage:"gRPC server listen address"`
	Network string `default:"tcp" validate:"oneof=tcp udp" usage:"gRPC server listen network: tpc/udp"`
}
