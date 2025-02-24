package southbound

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/gofrs/uuid/v5"
	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

// Override the unimplemented methods from core.go
func (sb *SouthboundService) CreateHardwareConfig(ctx context.Context, req *v1.CreateHardwareConfigRequest) (*v1.HardwareConfigResponse, error) {
	versionSetID, err := uuid.FromString(req.VersionSetId)
	if err != nil {
		return nil, fmt.Errorf("invalid version set ID: %w", err)
	}

	// Parse IP CIDR
	_, ipNet, err := net.ParseCIDR(req.IpCidr)
	if err != nil {
		return nil, fmt.Errorf("invalid IP CIDR: %w", err)
	}

	nodeID := int(req.NodeId)
	config := &types.HardwareConfig{
		NodeID:       &nodeID,
		Device:       req.Device,
		IPCIDR:       *ipNet,
		VersionSetID: &versionSetID,
		State:        types.VERSION_STATE_DRAFT,
		CreatedBy:    "system", // You might want to get this from context or auth
	}

	err = sb.db.CreateHwConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create hardware config: %w", err)
	}

	return &v1.HardwareConfigResponse{
		HardwareConfig: &v1.HardwareConfig{
			Id:           int32(config.ID),
			NodeId:       req.NodeId,
			Device:       config.Device,
			IpCidr:       config.IPCIDR.String(),
			VersionSetId: config.VersionSetID.String(),
		},
	}, nil
}

func (sb *SouthboundService) GetHardwareConfig(ctx context.Context, req *v1.GetHardwareConfigRequest) (*v1.HardwareConfigResponse, error) {
	config, err := sb.db.GetHwConfigPByID(ctx, int(req.Id))
	if err != nil {
		return nil, fmt.Errorf("failed to get hardware config: %w", err)
	}

	return &v1.HardwareConfigResponse{
		HardwareConfig: &v1.HardwareConfig{
			Id:           int32(config.ID),
			NodeId:       int32(*config.NodeID),
			Device:       config.Device,
			IpCidr:       config.IPCIDR.String(),
			VersionSetId: config.VersionSetID.String(),
		},
	}, nil
}

func (sb *SouthboundService) ListHardwareConfigs(ctx context.Context, req *v1.ListHardwareConfigsRequest) (*v1.ListHardwareConfigsResponse, error) {
	var configs []*types.HardwareConfig
	var err error

	if req.GetVersionSetId() != "" {
		versionSetID, err := uuid.FromString(req.GetVersionSetId())
		if err != nil {
			return nil, fmt.Errorf("invalid version set ID: %w", err)
		}
		configs, err = sb.db.GetHwConfigByVersionSetID(ctx, versionSetID)
	} else if req.GetNodeId() != 0 {
		nodeID := int(req.GetNodeId())
		configs, err = sb.db.GetHwConfigbyNodeID(ctx, nodeID)
	} else {
		// TODO: Implement GetAllHwConfigs in db package
		return nil, fmt.Errorf("listing all hardware configs not implemented")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list hardware configs: %w", err)
	}

	response := &v1.ListHardwareConfigsResponse{
		HardwareConfigs: make([]*v1.HardwareConfig, len(configs)),
	}

	for i, config := range configs {
		response.HardwareConfigs[i] = &v1.HardwareConfig{
			Id:           int32(config.ID),
			NodeId:       int32(*config.NodeID),
			Device:       config.Device,
			IpCidr:       config.IPCIDR.String(),
			VersionSetId: config.VersionSetID.String(),
		}
	}

	return response, nil
}

func (sb *SouthboundService) UpdateHardwareConfig(ctx context.Context, req *v1.UpdateHardwareConfigRequest) (*empty.Empty, error) {
	updates := make(map[string]interface{})

	if req.Device != nil {
		updates["device"] = *req.Device
	}
	if req.IpCidr != nil {
		// Parse IP CIDR
		_, ipNet, err := net.ParseCIDR(*req.IpCidr)
		if err != nil {
			return nil, fmt.Errorf("invalid IP CIDR: %w", err)
		}
		updates["ip_cidr"] = *ipNet
	}
	if req.VersionSetId != nil {
		versionSetID, err := uuid.FromString(*req.VersionSetId)
		if err != nil {
			return nil, fmt.Errorf("invalid version set ID: %w", err)
		}
		updates["version_set_id"] = versionSetID
	}

	err := sb.db.Update(ctx, "hardware_configs", updates, "id", strconv.Itoa(int(req.Id)))
	if err != nil {
		return nil, fmt.Errorf("failed to update hardware config: %w", err)
	}

	return &empty.Empty{}, nil
}

func (sb *SouthboundService) DeleteHardwareConfig(ctx context.Context, req *v1.DeleteHardwareConfigRequest) (*empty.Empty, error) {
	err := sb.db.Delete(ctx, "hardware_configs", "id", strconv.Itoa(int(req.Id)))
	if err != nil {
		return nil, fmt.Errorf("failed to delete hardware config: %w", err)
	}

	return &empty.Empty{}, nil
}
