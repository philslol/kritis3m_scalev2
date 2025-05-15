package southbound

import (
	"context"
	"time"

	grpc_controlplane "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/control_plane"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (sb *SouthboundService) TriggerCertReq(ctx context.Context, req *grpc_southbound.TriggerCertReqRequest) (*grpc_southbound.TriggerCertReqResponse, error) {
	ret := 0
	plane := req.CertType

	if plane != grpc_southbound.CertType_DATAPLANE && plane != grpc_southbound.CertType_CONTROLPLANE {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cert type: %v", plane)
	}
	if req.SerialNumber == "" {
		return nil, status.Errorf(codes.InvalidArgument, "serial number is required")
	}

	// in the future we make a db query and check if node exits, and we query est server address,host,port from db
	client, conn, err := getControlPlaneClient(sb.addr)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to get control plane client: %v", err)
	}
	defer conn.Close()

	//TODO: check if algo is supported by est server

	// TODO: get hostname and ip addr from db
	// the est server could be started now as a service with timeout awaiting the request from the control plane
	client.SendCertificateRequest(ctx, &grpc_controlplane.CertificateRequest{
		SerialNumber: req.SerialNumber,
		CertType:     req.CertType,
		HostName:     "example hostname",
		IpAddr:       "example ip",
		Algo:         req.Algo,
		AltAlgo:      req.AltAlgo,
		Port:         0,
	})

	return &grpc_southbound.TriggerCertReqResponse{Retcode: int32(ret)}, nil
}

func (sb *SouthboundService) TriggerFleetCertReq(ctx context.Context, req *grpc_southbound.TriggerFleetCertRequest) (*grpc_southbound.TriggerFleetCertReqResponse, error) {
	ret := 0

	var versionSetID *uuid.UUID
	if req.GetVersionSetId() != "" {
		id, err := uuid.FromString(req.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
		}
		versionSetID = &id
	} else {
		return nil, status.Errorf(codes.InvalidArgument, "version set id is required")
	}

	//TODO: check if version set id exists in db
	nodes, err := sb.db.ListNodes(ctx, versionSetID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list nodes: %v", err)
	}
	queryNodes := []*types.Node{}
	//filter nodes, where now() - last_seen > 2 minutes
	for _, node := range nodes {
		if time.Since(*node.LastSeen) < 2*time.Minute {
			queryNodes = append(queryNodes, node)
		}
	}

	// in the future we make a db query and check if node exits, and we query est server address,host,port from db
	client, conn, err := getControlPlaneClient(sb.addr)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to get control plane client: %v", err)
	}
	defer conn.Close()

	for _, node := range queryNodes {
		client.SendCertificateRequest(ctx, &grpc_controlplane.CertificateRequest{
			SerialNumber: node.SerialNumber,
			CertType:     req.CertType,
			HostName:     "example hostname",
			IpAddr:       "example ip",
			Algo:         req.Algo,
			AltAlgo:      req.AltAlgo,
			Port:         0,
		})
	}

	return &grpc_southbound.TriggerFleetCertReqResponse{Retcode: int32(ret)}, nil
}
