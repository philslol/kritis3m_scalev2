package southbound

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *SouthboundService) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest) (*v1.GroupResponse, error) {
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

func (s *SouthboundService) GetGroup(ctx context.Context, req *v1.GetGroupRequest) (*v1.GroupResponse, error) {
	var group *types.Group
	var err error

	switch req.Query.(type) {
	case *v1.GetGroupRequest_Id:
		group, err = s.db.GetByID(ctx, int(req.GetId()))
		if err != nil {
			return nil, fmt.Errorf("failed to get group: %v", err)
		}
	case *v1.GetGroupRequest_GroupQuery:
		versionID := uuid.FromStringOrNil(req.GetGroupQuery().GetVersionSetId())
		name := req.GetGroupQuery().GetGroupName()
		group, err = s.db.GetGroupByName(ctx, name, &versionID)

		if err != nil {
			return nil, fmt.Errorf("failed to get group: %v", err)
		}
	}

	return convertGroupToResponse(group), nil
}

func (s *SouthboundService) ListGroups(ctx context.Context, req *v1.ListGroupsRequest) (*v1.ListGroupsResponse, error) {
	var response *v1.ListGroupsResponse
	if req.VersionSetId != nil {
		versionID := uuid.FromStringOrNil(req.GetVersionSetId())

		groups, err := s.db.GetListGroup(ctx, &versionID)
		if err != nil {
			return nil, fmt.Errorf("failed to list groups: %v", err)
		}

		response = &v1.ListGroupsResponse{
			Groups: make([]*v1.GroupResponse, len(groups)),
		}

		for i, group := range groups {
			response.Groups[i] = convertGroupToResponse(group)
		}
	}

	return response, nil
}

func (s *SouthboundService) UpdateGroup(ctx context.Context, req *v1.UpdateGroupRequest) (*empty.Empty, error) {
	updates := make(map[string]interface{})

	var where_string string
	switch req.Query.(type) {
	case *v1.UpdateGroupRequest_Id:
		updates["id"] = int(req.GetId())
		where_string = fmt.Sprintf("id = %d", req.GetId())
	case *v1.UpdateGroupRequest_GroupQuery:
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

func (s *SouthboundService) DeleteGroup(ctx context.Context, req *v1.DeleteGroupRequest) (*empty.Empty, error) {
	if err := s.db.Delete(ctx, "groups", "id", fmt.Sprintf("%d", req.GetId())); err != nil {
		return nil, fmt.Errorf("failed to delete group: %v", err)
	}

	return &empty.Empty{}, nil
}

func convertGroupToResponse(dbGroup *types.Group) *v1.GroupResponse {
	protoGroup := &v1.Group{
		Id:                 int32(dbGroup.ID),
		Name:               dbGroup.Name,
		LogLevel:           int32(dbGroup.LogLevel),
		EndpointConfigName: dbGroup.EndpointConfigName,
		LegacyConfigName:   &dbGroup.LegacyConfigName,
		VersionSetId:       dbGroup.VersionSetID.String(),
	}

	return &v1.GroupResponse{
		Group: protoGroup,
	}
}

func getCurrentUser(ctx context.Context) string {
	// TODO: Implement proper user context extraction
	return "system"
}
