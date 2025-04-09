package southbound

import (
	"context"
	"fmt"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/gofrs/uuid/v5"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SouthboundService) CreateGroup(ctx context.Context, req *grpc_southbound.CreateGroupRequest) (*grpc_southbound.GroupResponse, error) {
	group := &types.Group{
		Name:               req.GetName(),
		LogLevel:           int(req.GetLogLevel()),
		CreatedBy:          req.GetCreatedBy(),
		EndpointConfigName: req.GetEndpointConfigName(),
		LegacyConfigName:   req.GetLegacyConfigName(),
		VersionSetID:       uuid.FromStringOrNil(req.GetVersionSetId()),
	}

	if err := s.db.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %v", err)
	}

	return convertGroupToResponse(group), nil
}

func (s *SouthboundService) GetGroup(ctx context.Context, req *grpc_southbound.GetGroupRequest) (*grpc_southbound.GroupResponse, error) {
	var group *types.Group
	var err error

	switch req.Query.(type) {
	case *grpc_southbound.GetGroupRequest_Id:
		group, err = s.db.GetByID(ctx, int(req.GetId()))
		if err != nil {
			return nil, fmt.Errorf("failed to get group: %v", err)
		}
	case *grpc_southbound.GetGroupRequest_GroupQuery:
		versionID := uuid.FromStringOrNil(req.GetGroupQuery().GetVersionSetId())
		name := req.GetGroupQuery().GetGroupName()
		group, err = s.db.GetGroupByName(ctx, name, &versionID)

		if err != nil {
			return nil, fmt.Errorf("failed to get group: %v", err)
		}
	}

	return convertGroupToResponse(group), nil
}

func (s *SouthboundService) ListGroups(ctx context.Context, req *grpc_southbound.ListGroupsRequest) (*grpc_southbound.ListGroupsResponse, error) {
	var response *grpc_southbound.ListGroupsResponse
	var versionSetID uuid.UUID
	var groups []*types.Group
	var err error
	if req.VersionSetId != nil {
		versionSetID = uuid.FromStringOrNil(req.GetVersionSetId())
		groups, err = s.db.GetListGroup(ctx, &versionSetID)
	} else {
		groups, err = s.db.GetListGroup(ctx, nil)
	}

	if err != nil {
		log.Error().Err(err).Msg("failed to list groups")
		return nil, status.Errorf(codes.Internal, "failed to list groups: %v", err)
	}

	response = &grpc_southbound.ListGroupsResponse{
		Groups: make([]*grpc_southbound.GroupResponse, len(groups)),
	}

	for i, group := range groups {
		response.Groups[i] = convertGroupToResponse(group)
	}

	return response, nil
}

func (s *SouthboundService) UpdateGroup(ctx context.Context, req *grpc_southbound.UpdateGroupRequest) (*empty.Empty, error) {
	updates := make(map[string]interface{})

	var where_string string
	switch req.Query.(type) {
	case *grpc_southbound.UpdateGroupRequest_Id:
		updates["id"] = int(req.GetId())
		where_string = fmt.Sprintf("id = %d", req.GetId())
	case *grpc_southbound.UpdateGroupRequest_GroupQuery:
		name := req.GetGroupQuery().GetGroupName()
		versionID := uuid.FromStringOrNil(req.GetGroupQuery().GetVersionSetId())
		where_string = fmt.Sprintf("name = %s AND version_set_id = %s", name, versionID)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid query")
	}
	if req.VersionSetId != nil {
		updates["version_set_id"] = uuid.FromStringOrNil(*req.VersionSetId)
	}
	if req.LogLevel != nil {
		updates["log_level"] = int(req.GetLogLevel())
	}
	if req.EndpointConfigName != nil {
		updates["endpoint_config_name"] = *req.EndpointConfigName
	}
	if req.LegacyConfigName != nil {
		updates["legacy_config_id"] = req.LegacyConfigName
	}
	if err := s.db.UpdateWhere(ctx, "groups", updates, where_string); err != nil {
		//internal error
		log.Error().Err(err).Msg("failed to update group")
		return nil, status.Errorf(codes.Internal, "failed to update group: %v", err)
	}
	return &empty.Empty{}, nil
}

func (s *SouthboundService) DeleteGroup(ctx context.Context, req *grpc_southbound.DeleteGroupRequest) (*empty.Empty, error) {
	if err := s.db.Delete(ctx, "groups", "id", fmt.Sprintf("%d", req.GetId())); err != nil {
		return nil, fmt.Errorf("failed to delete group: %v", err)
	}

	return &empty.Empty{}, nil
}

func convertGroupToResponse(dbGroup *types.Group) *grpc_southbound.GroupResponse {
	protoGroup := &grpc_southbound.Group{
		Id:                 int32(dbGroup.ID),
		Name:               dbGroup.Name,
		LogLevel:           int32(dbGroup.LogLevel),
		EndpointConfigName: dbGroup.EndpointConfigName,
		LegacyConfigName:   &dbGroup.LegacyConfigName,
		VersionSetId:       dbGroup.VersionSetID.String(),
	}

	return &grpc_southbound.GroupResponse{
		Group: protoGroup,
	}
}
