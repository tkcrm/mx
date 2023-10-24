package grpc_client_test

import (
	"testing"

	"github.com/tkcrm/mx/clients/grpc_client"
	"google.golang.org/grpc"
)

func TestNew(t *testing.T) {
	_, err := grpc_client.New[any](grpc_client.Config{}, nil, func(cc grpc.ClientConnInterface) any {
		return "as"
	})
	if err != nil {
		t.Fatal(err)
	}
}
