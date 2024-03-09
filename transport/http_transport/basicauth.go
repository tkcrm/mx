package http_transport

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net/http"
)

type BasicAuthConfig struct {
	Enabled  bool   `default:"false" usage:"allows to enable basic auth"`
	Username string `usage:"auth username"`
	Password string `usage:"auth password"`
}

func BasicAuthHandler(next http.Handler, cfg BasicAuthConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := checkBasicAuth(r, cfg); err != nil {
			w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func checkBasicAuth(r *http.Request, cfg BasicAuthConfig) error {
	if !cfg.Enabled {
		return nil
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		return fmt.Errorf("expected basic auth")
	}

	if username == "" || password == "" {
		return fmt.Errorf("empty username or password")
	}

	userEncoded := sha256.Sum256([]byte(username))
	passEncoded := sha256.Sum256([]byte(password))

	expectedUserEncoded := sha256.Sum256([]byte(cfg.Username))
	expectedPassEncoded := sha256.Sum256([]byte(cfg.Password))

	usernameMatch := (subtle.ConstantTimeCompare(userEncoded[:], expectedUserEncoded[:]) == 1)
	passwordMatch := (subtle.ConstantTimeCompare(passEncoded[:], expectedPassEncoded[:]) == 1)

	if usernameMatch && passwordMatch {
		return nil
	}

	return fmt.Errorf("credentials mismatch")
}
