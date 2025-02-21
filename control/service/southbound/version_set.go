package southbound

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (sb *SouthboundService) CreateVersionSet(ctx context.Context, req *v1.CreateVersionSetRequest) (*v1.VersionSetResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	// Convert protobuf metadata to JSON bytes
	metadataBytes, err := json.Marshal(req.GetMetadata())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
	}
	var rsp_description *string
	if description := req.GetDescription(); description == "" {
		rsp_description = nil
	} else {
		rsp_description = &description
	}

	vs := types.VersionSet{
		Name:        req.GetName(),
		Description: rsp_description,
		CreatedBy:   req.GetCreatedBy(),
		Metadata:    metadataBytes,
	}

	id, err := sb.db.CreateVersionSet(ctx, vs)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create version set: %v", err)
	}

	// Fetch the created version set to return complete data
	createdVS, err := sb.db.GetVersionSetByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "version set created but failed to fetch: %v", err)
	}

	return convertVersionSetToResponse(createdVS)
}

func (sb *SouthboundService) GetVersionSet(ctx context.Context, req *v1.GetVersionSetRequest) (*v1.VersionSetResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "version set ID is required")
	}

	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	vs, err := sb.db.GetVersionSetByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get version set: %v", err)
	}

	if vs == nil {
		return nil, status.Error(codes.NotFound, "version set not found")
	}

	return convertVersionSetToResponse(vs)
}

func (sb *SouthboundService) ListVersionSets(ctx context.Context, req *v1.ListVersionSetsRequest) (*v1.ListVersionSetsResponse, error) {
	versionSets, err := sb.db.ListVersionSets(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list version sets: %v", err)
	}

	response := &v1.ListVersionSetsResponse{}
	for _, vs := range versionSets {
		vsResp, err := convertVersionSetToResponse(vs)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert version set: %v", err)
		}
		response.VersionSets = append(response.VersionSets, vsResp.GetVersionSet())
	}

	return response, nil
}

func (sb *SouthboundService) UpdateVersionSet(ctx context.Context, req *v1.UpdateVersionSetRequest) (*empty.Empty, error) {
	if req == nil || req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "version set ID is required")
	}

	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	// Build updates map with non-nil fields
	updates := make(map[string]interface{})
	if name := req.GetName(); name != "" {
		updates["name"] = name
	}
	if description := req.GetDescription(); description != "" {
		updates["description"] = description
	}
	if metadata := req.GetMetadata(); metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %v", err)
		}
		updates["metadata"] = metadataBytes
	}

	if len(updates) == 0 {
		return &empty.Empty{}, nil
	}

	err = sb.db.Update(ctx, "version_sets", updates, "id", id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update version set: %v", err)
	}

	return &empty.Empty{}, nil
}

func (sb *SouthboundService) DeleteVersionSet(ctx context.Context, req *v1.DeleteVersionSetRequest) (*empty.Empty, error) {
	if req == nil || req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "version set ID is required")
	}

	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	err = sb.db.Delete(ctx, "version_sets", "id", id.String())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete version set: %v", err)
	}

	return &empty.Empty{}, nil
}

func (sb *SouthboundService) ActivateVersionSet(ctx context.Context, req *v1.ActivateVersionSetRequest) (*v1.VersionSetResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "version set ID is required")
	}

	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	updates := map[string]interface{}{
		"state": "active",
	}

	err = sb.db.Update(ctx, "version_sets", updates, "id", id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to activate version set: %v", err)
	}

	// Fetch the activated version set to return complete data
	vs, err := sb.db.GetVersionSetByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "version set activated but failed to fetch: %v", err)
	}

	return convertVersionSetToResponse(vs)
}

func (sb *SouthboundService) DisableVersionSet(ctx context.Context, req *v1.DisableVersionSetRequest) (*v1.VersionSetResponse, error) {
	if req == nil || req.GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "version set ID is required")
	}

	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	updates := map[string]interface{}{
		"state": "disabled",
	}

	err = sb.db.Update(ctx, "version_sets", updates, "id", id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to disable version set: %v", err)
	}

	// Fetch the disabled version set to return complete data
	vs, err := sb.db.GetVersionSetByID(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "version set disabled but failed to fetch: %v", err)
	}

	return convertVersionSetToResponse(vs)
}

// Helper function to convert a VersionSet to a VersionSetResponse
func convertVersionSetToResponse(vs *types.VersionSet) (*v1.VersionSetResponse, error) {
	if vs == nil {
		return nil, fmt.Errorf("version set is nil")
	}

	// Parse metadata JSON back to protobuf struct
	var rawMetadata map[string]interface{}
	if err := json.Unmarshal(vs.Metadata, &rawMetadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	metadata, err := structpb.NewStruct(rawMetadata)
	if err != nil {
		return nil, fmt.Errorf("failed to convert metadata to protobuf struct: %w", err)
	}

	var activatedAt, disabledAt *timestamppb.Timestamp
	if vs.ActivatedAt != nil && !vs.ActivatedAt.IsZero() {
		activatedAt = timestamppb.New(*vs.ActivatedAt)
	}
	if vs.DisabledAt != nil && !vs.DisabledAt.IsZero() {
		disabledAt = timestamppb.New(*vs.DisabledAt)
	}

	return &v1.VersionSetResponse{
		VersionSet: &v1.VersionSet{
			Id:          vs.ID.String(),
			Name:        vs.Name,
			Description: *vs.Description,
			State:       v1.VersionState(v1.VersionState_value[vs.State]),
			CreatedBy:   vs.CreatedBy,
			ActivatedAt: activatedAt,
			DisabledAt:  disabledAt,
			Metadata:    metadata,
		},
	}, nil
}
