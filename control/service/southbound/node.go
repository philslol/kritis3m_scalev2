package southbound

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/gofrs/uuid"
	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

func (sb *SouthboundService) CreateNode(ctx context.Context, req *v1.CreateNodeRequest) (*v1.NodeResponse, error) {
	//create locality out of req in the database
	var node types.Node
	node.SerialNumber = req.GetSerialNumber()
	node.NetworkIndex = int(req.GetNetworkIndex())
	node.Locality = req.GetLocality()
	//convert versionSetid string to uuid
	uuid_version, err := uuid.FromString(req.GetVersionSetId())
	node.VersionSetID = &uuid_version
	if err != nil {
		return nil, fmt.Errorf("invalid UUID: %w", err)
	}

	new_node, err := sb.db.CreateNode(ctx, &node)
	if err != nil {
		log.Err(err).Msg("failed to create node")
		return nil, err
	}

	nodeResponse := &v1.Node{
		Id:           int32(new_node.ID),
		SerialNumber: new_node.SerialNumber,
		NetworkIndex: int32(new_node.NetworkIndex),
		Locality:     new_node.Locality,
		VersionSetId: new_node.VersionSetID.String(),
	}

	return &v1.NodeResponse{
		Node: nodeResponse,
	}, nil
}
func (sb *SouthboundService) GetNode(ctx context.Context, req *v1.GetNodeRequest) (*v1.NodeResponse, error) {
	//get node from database
	node, err := sb.db.GetNode(ctx, int(req.GetId()))
	if err != nil {
		log.Err(err).Msg("failed to get node")
		return nil, err
	}

	nodeResponse := &v1.Node{
		Id:           int32(node.ID),
		SerialNumber: node.SerialNumber,
		NetworkIndex: int32(node.NetworkIndex),
		Locality:     node.Locality,
		VersionSetId: node.VersionSetID.String(),
	}
	var node_rsp v1.NodeResponse
	node_rsp.Node = nodeResponse

	if req.GetInclude() {
		//get hardware configs of the node
		hw_configs, err := sb.db.GetHwConfigbyNodeID(ctx, int(node.ID))
		if err != nil {
			log.Err(err).Msg("failed to get hardware configs, query will be executed anyway")
			return nil, err
		}
		node_rsp.HwConfigs = make([]*v1.HardwareConfig, len(hw_configs))
		for i, hw_config := range hw_configs {
			node_rsp.HwConfigs[i] = &v1.HardwareConfig{
				Id:           int32(hw_config.ID),
				NodeId:       int32(*hw_config.NodeID),
				Device:       hw_config.Device,
				IpCidr:       hw_config.IPCIDR.String(),
				VersionSetId: hw_config.VersionSetID.String(),
			}
		}
		proxies, err := sb.db.GetProxiesByNodeID(ctx, int(node.ID))
		if err != nil {
			log.Err(err).Msg("failed to get proxies, query will be executed anyway")
		}

		node_rsp.Proxy = make([]*v1.Proxy, len(proxies))
		//convert proxies to v1.Proxy
		for i, proxy := range proxies {
			node_rsp.Proxy[i] = &v1.Proxy{
				Id:                 int32(proxy.ID),
				NodeId:             int32(*proxy.NodeID),
				GroupId:            int32(*proxy.GroupID),
				State:              proxy.State,
				ProxyType:          v1.ProxyType(types.ProxyTypeMap[proxy.ProxyType]),
				ServerEndpointAddr: proxy.ServerEndpointAddr,
				ClientEndpointAddr: proxy.ClientEndpointAddr,
				VersionSetId:       proxy.VersionSetID.String(),
				CreatedBy:          proxy.CreatedBy,
			}
		}

	} else {
		node_rsp.Proxy = nil
		node_rsp.HwConfigs = nil
	}

	return &node_rsp, nil
}

func (sb *SouthboundService) UpdateNode(ctx context.Context, req *v1.UpdateNodeRequest) (*v1.NodeResponse, error) {
	//update node in the database
	var node types.Node = types.Node{
		ID:           int(req.GetId()),
		SerialNumber: req.GetSerialNumber(),
		NetworkIndex: int(req.GetNetworkIndex()),
		Locality:     req.GetLocality(),
	}
	//convert versionSetid string to uuid
	uuid_version, err := uuid.FromString(req.GetVersionSetId())
	if err != nil {
		log.Err(err).Msg("failed to convert versionSetId to uuid")
		return nil, err
	}
	node.VersionSetID = &uuid_version

	new_node, err := sb.db.UpdateNode(ctx, &node)

	if err != nil {
		log.Err(err).Msg("failed to update node")
		return nil, err
	}

	return &v1.NodeResponse{
		Node: &v1.Node{
			Id: int32(new_node.ID),
		},
	}, nil
}

func (sb *SouthboundService) DeleteNode(ctx context.Context, req *v1.DeleteNodeRequest) (empty.Empty, error) {
	err := sb.db.DeleteNode(ctx, int(req.GetId()))
	if err != nil {
		log.Err(err).Msg("failed to delete node")
		return emptypb.Empty{}, err
	}
	return empty.Empty{}, nil
}
