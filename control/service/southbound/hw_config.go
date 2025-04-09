package southbound

import (
	"context"
	"fmt"
	"strconv"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/gofrs/uuid/v5"
	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

// Override the unimplemented methods from core.go
func (sb *SouthboundService) CreateHardwareConfig(ctx context.Context, req *grpc_southbound.CreateHardwareConfigRequest) (*grpc_southbound.HardwareConfigResponse, error) {
	versionSetID, err := uuid.FromString(req.VersionSetId)
	if err != nil {
		return nil, fmt.Errorf("invalid version set ID: %w", err)
	}

	config := &types.HardwareConfig{
		NodeSerial:   req.NodeSerialNumber,
		Device:       req.Device,
		IPCIDR:       req.IpCidr,
		VersionSetID: versionSetID,
		CreatedBy:    "system", // You might want to get this from context or auth
	}

	err = sb.db.CreateHwConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create hardware config: %w", err)
	}

	return &grpc_southbound.HardwareConfigResponse{
		HardwareConfig: []*grpc_southbound.HardwareConfig{
			{
				Id:               int32(config.ID),
				NodeSerialNumber: config.NodeSerial,
				Device:           config.Device,
				IpCidr:           config.IPCIDR,
				VersionSetId:     config.VersionSetID.String(),
			},
		},
	}, nil
}

func (sb *SouthboundService) GetHardwareConfig(ctx context.Context, req *grpc_southbound.GetHardwareConfigRequest) (*grpc_southbound.HardwareConfigResponse, error) {
	var configs []*types.HardwareConfig

	response := &grpc_southbound.HardwareConfigResponse{
		HardwareConfig: []*grpc_southbound.HardwareConfig{},
	}

	switch req.Query.(type) {
	case *grpc_southbound.GetHardwareConfigRequest_Id:
		id := int(req.GetId())
		config, err := sb.db.GetHwConfigPByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("failed to get hardware config: %w", err)
		}
		configs = append(configs, config)
	case *grpc_southbound.GetHardwareConfigRequest_HardwareConfigQuery:
		versionSetID, err := uuid.FromString(req.GetHardwareConfigQuery().VersionSetId)
		if err != nil {
			return nil, fmt.Errorf("failed to get hardware config: %w", err)
		}
		serialNumber := req.GetHardwareConfigQuery().NodeSerialNumber

		configs, err = sb.db.GetHwConfigBySerial(ctx, serialNumber, versionSetID)
		if err != nil {
			return nil, fmt.Errorf("failed to get hardware config: %w", err)
		}

	case *grpc_southbound.GetHardwareConfigRequest_VersionSetId:
		versionSetID, err := uuid.FromString(req.GetVersionSetId())
		if err != nil {
			return nil, fmt.Errorf("failed to get hardware config: %w", err)
		}
		configs, err = sb.db.GetHwConfigByVersionSetID(ctx, versionSetID)
		if err != nil {
			return nil, fmt.Errorf("failed to get hardware config: %w", err)
		}
	}
	response.HardwareConfig = make([]*grpc_southbound.HardwareConfig, len(configs))

	for i, config := range configs {
		response.HardwareConfig[i] = &grpc_southbound.HardwareConfig{
			Id:               int32(config.ID),
			NodeSerialNumber: config.NodeSerial,
			Device:           config.Device,
			IpCidr:           config.IPCIDR,
			VersionSetId:     config.VersionSetID.String(),
		}
	}

	return response, nil
}

func (sb *SouthboundService) UpdateHardwareConfig(ctx context.Context, req *grpc_southbound.UpdateHardwareConfigRequest) (*empty.Empty, error) {
	updates := make(map[string]interface{})

	if req.Device != nil {
		updates["device"] = *req.Device
	}
	if req.IpCidr != nil {
		updates["ip_cidr"] = *req.IpCidr
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

func (sb *SouthboundService) DeleteHardwareConfig(ctx context.Context, req *grpc_southbound.DeleteHardwareConfigRequest) (*empty.Empty, error) {
	err := sb.db.Delete(ctx, "hardware_configs", "id", strconv.Itoa(int(req.Id)))
	if err != nil {
		return nil, fmt.Errorf("failed to delete hardware config: %w", err)
	}

	return &empty.Empty{}, nil
}
