package southbound

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

func (s *SouthboundService) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest) (*v1.GroupResponse, error) {
	group := &types.Group{
		Name:     req.GetName(),
		LogLevel: int(req.GetLogLevel()),
	}

	if req.EndpointConfigId != 0 {
		configID := int(req.EndpointConfigId)
		group.EndpointConfigID = &configID
	}

	if req.VersionSetId != "" {
		version := uuid.FromStringOrNil(req.VersionSetId)
		group.VersionSetID = &version
	}

	if req.LegacyConfigId != nil {
		legacyID := int(*req.LegacyConfigId)
		group.LegacyConfigID = &legacyID
	}

	if err := s.db.CreateGroup(ctx, group); err != nil {
		return nil, fmt.Errorf("failed to create group: %v", err)
	}

	return convertGroupToResponse(group), nil
}

func (s *SouthboundService) GetGroup(ctx context.Context, req *v1.GetGroupRequest) (*v1.GroupResponse, error) {
	group, err := s.db.GetByID(ctx, int(req.GetId()))
	if err != nil {
		return nil, fmt.Errorf("failed to get group: %v", err)
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

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	updates["log_level"] = int(req.GetLogLevel())
	if req.EndpointConfigId != nil {
		configID := int(*req.EndpointConfigId)
		updates["endpoint_config_id"] = &configID
	}
	if req.LegacyConfigId != nil {
		legacyID := int(*req.LegacyConfigId)
		updates["legacy_config_id"] = &legacyID
	}

	if req.VersionSetId != nil {
		version := uuid.FromStringOrNil(*req.VersionSetId)
		updates["legacy_config_id"] = version
	}

	if err := s.db.Update(ctx, "groups", updates, "id", int(req.GetId())); err != nil {
		return nil, fmt.Errorf("failed to update group: %v", err)
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
		Id:       int32(dbGroup.ID),
		Name:     dbGroup.Name,
		LogLevel: int32(dbGroup.LogLevel),
	}

	if dbGroup.EndpointConfigID != nil {
		protoGroup.EndpointConfigId = int32(*dbGroup.EndpointConfigID)
	}

	if dbGroup.LegacyConfigID != nil {
		legacyID := int32(*dbGroup.LegacyConfigID)
		protoGroup.LegacyConfigId = &legacyID
	}

	return &v1.GroupResponse{
		Group: protoGroup,
	}
}

func getCurrentUser(ctx context.Context) string {
	// TODO: Implement proper user context extraction
	return "system"
}
