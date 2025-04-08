package ops

import (
	"net/http"

	"github.com/tkcrm/mx/transport/http_transport"
)

type opsService interface {
	Name() string
	getEnabled() bool
	getPort() string
	getHTTPOptions() []http_transport.Option
	initService(mux *http.ServeMux)
}
