package southbound

import (
	"context"
	"os"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	grpc "google.golang.org/grpc"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/credentials/insecure"
)

func getClient(address string, ctx context.Context) (v1.ControlPlaneClient, *grpc.ClientConn, error) {

	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	conn, err := grpc.Dial(address, grpcOptions...)
	if err != nil {
		log.Fatal().Caller().Err(err).Msgf("Could not connect: %v", err)
		os.Exit(-1) // we get here if logging is suppressed (i.e., json output)
	}

	client := v1.NewControlPlaneClient(conn)
	return client, conn, nil

}

func (sb *SouthboundService) ActivateFleet(ctx context.Context, req *v1.ActivateFleetRequest) (*v1.ActivateResponse, error) {

	client, conn, err := getClient(sb.addr, ctx)
	if err != nil {
		log.Err(err).Msg("Could not get client")
		return nil, err
	}
	return nil, nil
}
func (sb *SouthboundService) ActivateNode(ctx context.Context, req *v1.ActivateNodeRequest) (*v1.ActivateResponse, error) {

	// check arguments
	if req.SerialNumber == "" || req.VersionSetId == "" {
		return nil, status.Error(codes.InvalidArgument, "SerialNumber and VersionSetId are required")
	}

	client, conn, err := getClient(sb.addr, ctx)
	if err != nil {
		log.Err(err).Msg("Could not get client")
		return nil, err
	}
	update_node := &v1.NodeUpdateItem{
		SerialNumber:     req.SerialNumber,
		NetworkIndex:     0,
		Locality:         0,
		VersionSetId:     req.VersionSetId,
		GroupProxyUpdate: nil,
	}
	update_node.SerialNumber = req.SerialNumber
	update_node.VersionSetId = req.VersionSetId
	client.UpdateNode(ctx, req)

	return nil, nil

}
