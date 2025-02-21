package cli

import (
	"context"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"google.golang.org/grpc"
)

var (
	testClient v1.SouthboundClient
)

// SetTestClient sets a mock client for testing
func SetTestClient(client v1.SouthboundClient) {
	testClient = client
}

// createClient returns a gRPC client for the node service
func createClient() (context.Context, v1.SouthboundClient, *grpc.ClientConn, context.CancelFunc, error) {
	if testClient != nil {
		return context.Background(), testClient, nil, func() {}, nil
	}

	// Your existing client creation logic here
	// This is just a placeholder - replace with your actual implementation
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		return nil, nil, nil, nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	client := v1.NewSouthboundClient(conn)

	return ctx, client, conn, cancel, nil
}
