package southbound

import (
	"context"

	grpc_controlplane "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/control_plane"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
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

	// TODO: get hostname and ip addr from db
	// the est server could be started now as a service with timeout awaiting the request from the control plane
	client.SendCertificateRequest(ctx, &grpc_controlplane.CertificateRequest{
		SerialNumber: req.SerialNumber,
		CertType:     req.CertType,
		HostName:     "example hostname",
		IpAddr:       "example ip",
		Port:         0,
	})

	return &grpc_southbound.TriggerCertReqResponse{Retcode: int32(ret)}, nil
}
