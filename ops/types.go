package ops

import "net/http"

type opsService interface {
	Name() string
	getEnabled() bool
	getPort() string
	initService(mux *http.ServeMux)
}
