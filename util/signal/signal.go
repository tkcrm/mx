package signal

import (
	"os"
	"syscall"
)

// Shutdown singals returns all the signals that are being watched for to shut down services.
func Shutdown() []os.Signal {
	return []os.Signal{
		syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL,
	}
}
