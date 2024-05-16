package connectrpc_client_test

import (
	"testing"

	"connectrpc.com/connect"
	"github.com/tkcrm/mx/clients/connectrpc_client"
)

func TestNew(t *testing.T) {
	_, err := connectrpc_client.New(connectrpc_client.Config{
		Name: "test-client",
	}, nil, func(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) any {
		return "as"
	})
	if err != nil {
		t.Fatal(err)
	}
}
