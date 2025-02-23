package southbound

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
)

func (sb *SouthboundService) ListEndpointConfigs(ctx context.Context, req *v1.ListEndpointConfigsRequest) (*v1.ListEndpointConfigsResponse, error) {
	log.Debug().Msgf("Listing endpoint configs")
	configs, err := sb.db.ListEndpointConfigs(ctx)
	if err != nil {
		log.Err(err).Msgf("Failed to list endpoint configs")
		return nil, err
	}

	response := &v1.ListEndpointConfigsResponse{
		Configs: make([]*v1.EndpointConfig, len(configs)),
	}

	for i, config := range configs {
		var versionSetId string
		if config.VersionSetID != nil {
			versionSetId = *config.VersionSetID
		}
		response.Configs[i] = &v1.EndpointConfig{
			Id:                   int32(config.ID),
			Name:                 config.Name,
			MutualAuth:           config.MutualAuth,
			NoEncryption:         config.NoEncryption,
			AslKeyExchangeMethod: v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[config.ASLKeyExchangeMethod]),
			Cipher:               config.Cipher,
			VersionSetId:         versionSetId,
			CreatedBy:            config.CreatedBy,
		}
	}

	return response, nil
}

func (sb *SouthboundService) CreateEndpointConfig(ctx context.Context, req *v1.CreateEndpointConfigRequest) (*v1.EndpointConfig, error) {
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
		config.VersionSetID = &req.VersionSetId
	}

	err := sb.db.CreateEndpointConfig(ctx, config)
	if err != nil {
		log.Err(err)
		return nil, err
	}

	var versionSetId string
	if config.VersionSetID != nil {
		versionSetId = *config.VersionSetID
	}

	return &v1.EndpointConfig{
		Id:                   int32(config.ID),
		Name:                 config.Name,
		MutualAuth:           config.MutualAuth,
		NoEncryption:         config.NoEncryption,
		AslKeyExchangeMethod: v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[config.ASLKeyExchangeMethod]),
		Cipher:               config.Cipher,
		VersionSetId:         versionSetId,
		CreatedBy:            config.CreatedBy,
	}, nil
}

func (sb *SouthboundService) GetEndpointConfig(ctx context.Context, req *v1.GetEndpointConfigRequest) (*v1.EndpointConfig, error) {
	config, err := sb.db.GetEndpointConfigByID(ctx, int(req.Id))
	if err != nil {
		log.Err(err)
		return nil, err
	}

	var versionSetId string
	if config.VersionSetID != nil {
		versionSetId = *config.VersionSetID
	}

	return &v1.EndpointConfig{
		Id:                   int32(config.ID),
		Name:                 config.Name,
		MutualAuth:           config.MutualAuth,
		NoEncryption:         config.NoEncryption,
		AslKeyExchangeMethod: v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[config.ASLKeyExchangeMethod]),
		Cipher:               config.Cipher,
		VersionSetId:         versionSetId,
		State:                config.State,
		CreatedBy:            config.CreatedBy,
	}, nil
}

func (sb *SouthboundService) UpdateEndpointConfig(ctx context.Context, req *v1.UpdateEndpointConfigRequest) (*emptypb.Empty, error) {
	updates := make(map[string]interface{})

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
	if req.VersionSetId != "" {
		updates["version_set_id"] = req.VersionSetId
	}

	err := sb.db.Update(ctx, "endpoint_configs", updates, "id", int(req.Id))
	if err != nil {
		log.Err(err)
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (sb *SouthboundService) DeleteEndpointConfig(ctx context.Context, req *v1.DeleteEndpointConfigRequest) (*emptypb.Empty, error) {
	err := sb.db.Delete(ctx, "endpoint_configs", "id", fmt.Sprintf("%d", req.Id))
	if err != nil {
		return nil, fmt.Errorf("failed to delete endpoint config: %w", err)
	}

	return &emptypb.Empty{}, nil
}
