package southbound

import (
	"context"

	grpc_est "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/est"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (sb *SouthboundService) EnrollCall(ctx context.Context, req *grpc_est.EnrollCallRequest) (*grpc_est.EnrollCallResponse, error) {
	// Convert v1.EnrollCallRequest to types.EnrollCallRequest
	issuedAt := req.IssuedAt.AsTime()
	expiresAt := req.ExpiresAt.AsTime()

	enrollReq := &types.EnrollCallRequest{
		EstSerialNumber:    req.EstSerialNumber,
		SerialNumber:       req.SerialNumber,
		Organization:       req.Organization,
		IssuedAt:           &issuedAt,
		ExpiresAt:          &expiresAt,
		SignatureAlgorithm: req.SignatureAlgorithm,
		Plane:              req.Plane,
	}

	// Store in database
	err := sb.db.CreateEnroll(ctx, enrollReq)
	if err != nil {
		return nil, err
	}

	// Return success response
	return &grpc_est.EnrollCallResponse{
		Retval: 0,
	}, nil
}
