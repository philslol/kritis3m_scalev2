package southbound

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
)

func (sb *SouthboundService) ListEndpointConfigs(ctx context.Context, req *grpc_southbound.ListEndpointConfigsRequest) (*grpc_southbound.ListEndpointConfigsResponse, error) {
	var versionSetID uuid.UUID
	var configs []*types.EndpointConfig
	var err error

	if req.VersionSetId != nil {
		versionSetID = uuid.FromStringOrNil(*req.VersionSetId)

		configs, err = sb.db.ListEndpointConfigs(ctx, &versionSetID)
	} else {
		configs, err = sb.db.ListEndpointConfigs(ctx, nil)
	}

	if err != nil {
		log.Err(err).Msgf("Failed to list endpoint configs")
		return nil, err
	}

	response := &grpc_southbound.ListEndpointConfigsResponse{
		Configs: make([]*grpc_southbound.EndpointConfig, len(configs)),
	}

	for i, config := range configs {
		versionSetId := config.VersionSetID.String()
		response.Configs[i] = &grpc_southbound.EndpointConfig{
			Id:                   int32(config.ID),
			Name:                 config.Name,
			MutualAuth:           config.MutualAuth,
			NoEncryption:         config.NoEncryption,
			AslKeyExchangeMethod: config.ASLKeyExchangeMethod,
			Cipher:               config.Cipher,
			VersionSetId:         versionSetId,
			CreatedBy:            config.CreatedBy,
		}
	}

	return response, nil
}

func (sb *SouthboundService) CreateEndpointConfig(ctx context.Context, req *grpc_southbound.CreateEndpointConfigRequest) (*grpc_southbound.EndpointConfig, error) {
	config := &types.EndpointConfig{
		Name:                 req.Name,
		MutualAuth:           req.MutualAuth,
		NoEncryption:         req.NoEncryption,
		ASLKeyExchangeMethod: req.AslKeyExchangeMethod.String(),
		Cipher:               req.Cipher,
		CreatedBy:            req.CreatedBy,
	}
	log.Debug().Msgf("Creating endpoint config: %v", config)

	if req.VersionSetId != "" {
		config.VersionSetID = uuid.FromStringOrNil(req.VersionSetId)
	}

	err := sb.db.CreateEndpointConfig(ctx, config)
	if err != nil {
		log.Err(err)
		return nil, err
	}

	return &grpc_southbound.EndpointConfig{
		Id:                   int32(config.ID),
		Name:                 config.Name,
		MutualAuth:           config.MutualAuth,
		NoEncryption:         config.NoEncryption,
		AslKeyExchangeMethod: config.ASLKeyExchangeMethod,
		Cipher:               config.Cipher,
		VersionSetId:         req.VersionSetId,
		CreatedBy:            config.CreatedBy,
	}, nil
}

func (sb *SouthboundService) GetEndpointConfig(ctx context.Context, req *grpc_southbound.GetEndpointConfigRequest) (*grpc_southbound.EndpointConfig, error) {
	//implement both query options:
	//1. GetEndpointConfigRequest_Id
	//2. GetEndpointConfigRequest_EndpointConfigQuery
	var config *types.EndpointConfig
	var err error

	switch req.Query.(type) {
	case *grpc_southbound.GetEndpointConfigRequest_Id:
		config, err = sb.db.GetEndpointConfigByID(ctx, int(req.GetId()))
		if err != nil {
			log.Err(err)
			return nil, err
		}
	case *grpc_southbound.GetEndpointConfigRequest_EndpointConfigQuery:
		versionSetID := uuid.FromStringOrNil(req.GetEndpointConfigQuery().VersionSetId)
		config, err = sb.db.GetEndpointConfigByName(ctx, req.GetEndpointConfigQuery().Name, &versionSetID)
		if err != nil {
			log.Err(err)
			return nil, err
		}
	}

	return &grpc_southbound.EndpointConfig{
		Id:                   int32(config.ID),
		Name:                 config.Name,
		MutualAuth:           config.MutualAuth,
		NoEncryption:         config.NoEncryption,
		AslKeyExchangeMethod: config.ASLKeyExchangeMethod,
		Cipher:               config.Cipher,
		VersionSetId:         config.VersionSetID.String(),
		CreatedBy:            config.CreatedBy,
	}, nil
}

func (sb *SouthboundService) UpdateEndpointConfig(ctx context.Context, req *grpc_southbound.UpdateEndpointConfigRequest) (*emptypb.Empty, error) {
	updates := make(map[string]interface{})
	var where_string string

	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.MutualAuth != nil {
		updates["mutual_auth"] = *req.MutualAuth
	}
	if req.NoEncryption != nil {
		updates["no_encryption"] = *req.NoEncryption
	}
	if req.AslKeyExchangeMethod != nil {
		updates["asl_key_exchange_method"] = req.AslKeyExchangeMethod.String()
	}
	if req.Cipher != nil {
		updates["cipher"] = *req.Cipher
	}

	switch req.Query.(type) {
	case *grpc_southbound.UpdateEndpointConfigRequest_Id:
		where_string := fmt.Sprintf("id = %d", req.GetId())
		err := sb.db.UpdateWhere(ctx, "endpoint_configs", updates, where_string)
		if err != nil {
			log.Err(err)
			return nil, err
		}
	case *grpc_southbound.UpdateEndpointConfigRequest_EndpointConfigQuery:
		name := req.GetEndpointConfigQuery().Name
		versionSetID := uuid.FromStringOrNil(req.GetEndpointConfigQuery().VersionSetId)
		where_string = fmt.Sprintf("name = %s AND version_set_id = %s", name, versionSetID)
	}

	err := sb.db.UpdateWhere(ctx, "endpoint_configs", updates, where_string)
	if err != nil {
		log.Err(err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (sb *SouthboundService) DeleteEndpointConfig(ctx context.Context, req *grpc_southbound.DeleteEndpointConfigRequest) (*emptypb.Empty, error) {
	err := sb.db.Delete(ctx, "endpoint_configs", "id", fmt.Sprintf("%d", req.Id))
	if err != nil {
		return nil, fmt.Errorf("failed to delete endpoint config: %w", err)
	}

	return &emptypb.Empty{}, nil
}
