package connectrpc_client_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/tkcrm/mx/clients/connectrpc_client"
)

func newStub(connect.HTTPClient, string, ...connect.ClientOption) string { return "stub" }

func TestNew(t *testing.T) {
	client, err := connectrpc_client.New(connectrpc_client.Config{
		Name: "test-client",
		Addr: "http://localhost:8080",
	}, nil, newStub)
	if err != nil {
		t.Fatal(err)
	}
	if client != "stub" {
		t.Fatalf("client = %q; want %q", client, "stub")
	}
}

func TestNew_errors(t *testing.T) {
	tests := map[string]struct {
		config connectrpc_client.Config
		fn     func(connect.HTTPClient, string, ...connect.ClientOption) string
	}{
		"nil init func": {config: connectrpc_client.Config{Name: "test-client", Addr: "http://localhost:8080"}},
		"empty name":    {config: connectrpc_client.Config{Addr: "http://localhost:8080"}, fn: newStub},
		"empty address": {config: connectrpc_client.Config{Name: "test-client"}, fn: newStub},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := connectrpc_client.New(tt.config, nil, tt.fn); err == nil {
				t.Fatal("New() = nil error; want an error")
			}
		})
	}
}
