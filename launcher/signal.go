package launcher

import (
	"os"
	"syscall"
)

// ShutdownSiganl returns all the signals that are being watched for to shut down services.
func ShutdownSiganl() []os.Signal {
	return []os.Signal{
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT,
	}
}
