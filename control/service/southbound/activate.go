package southbound

import (
	"context"
	"io"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func (sb *SouthboundService) getControlPlaneClient(ctx context.Context) (v1.ControlPlaneClient, *grpc.ClientConn, error) {
	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	conn, err := grpc.NewClient(sb.addr, grpcOptions...)
	if err != nil {
		log.Error().Err(err).Msg("Could not connect to control plane")
		return nil, nil, status.Error(codes.Internal, "Failed to connect to control plane")
	}

	client := v1.NewControlPlaneClient(conn)
	return client, conn, nil
}

func (sb *SouthboundService) ActivateFleet(ctx context.Context, req *v1.ActivateFleetRequest) (*v1.ActivateResponse, error) {
	if req.VersionSetId == "" {
		return nil, status.Error(codes.InvalidArgument, "VersionSetId is required")
	}

	// Get client
	client, conn, err := sb.getControlPlaneClient(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Send fleet update request
	stream, err := client.UpdateFleet(ctx, &v1.FleetUpdate{
		Transaction: &v1.Transaction{
			TxId: req.VersionSetId,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to update fleet")
		return nil, err
	}
	var resp *v1.UpdateResponse

	for {
		resp, err = stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error().Err(err).Msg("Failed to receive response")
			return nil, err
		}

	}

	return &v1.ActivateResponse{
		Retcode: int32(resp.UpdateState),
	}, nil
}

func (sb *SouthboundService) ActivateNode(ctx context.Context, req *v1.ActivateNodeRequest) (*v1.ActivateResponse, error) {
	// Check arguments
	if req.SerialNumber == "" || req.VersionSetId == "" {
		return nil, status.Error(codes.InvalidArgument, "SerialNumber and VersionSetId are required")
	}

	// Get node update from database
	nodeUpdate, err := sb.db.NodeUpdate(req.SerialNumber, req.VersionSetId, ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get node update from database")
		return nil, status.Error(codes.Internal, "Failed to get node update")
	}
	if nodeUpdate == nil {
		return nil, status.Error(codes.NotFound, "Node not found or no updates available")
	}

	// Get client and send update
	client, conn, err := sb.getControlPlaneClient(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Create the node update request
	update := &v1.NodeUpdate{
		NodeUpdateItem: nodeUpdate,
		Transaction: &v1.Transaction{
			TxId: req.VersionSetId,
		},
	}

	stream, err := client.UpdateNode(ctx, update)
	if err != nil {
		log.Error().Err(err).Msg("Failed to update node")
		return nil, err
	}

	// Get the first response from the stream
	resp, err := stream.Recv()
	if err != nil {
		log.Error().Err(err).Msg("Failed to receive response")
		return nil, err
	}

	return &v1.ActivateResponse{
		Retcode: int32(resp.UpdateState),
	}, nil
}
