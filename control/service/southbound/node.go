package southbound

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"
)

func (sb *SouthboundService) ListNodes(ctx context.Context, req *grpc_southbound.ListNodesRequest) (*grpc_southbound.ListNodesResponse, error) {
	var versionSetID *uuid.UUID
	if req.GetVersionSetId() != "" {
		id, err := uuid.FromString(req.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
		}
		versionSetID = &id
	}

	nodes, err := sb.db.ListNodes(ctx, versionSetID)
	if err != nil {
		log.Err(err).Msg("failed to list nodes")
		return nil, status.Error(codes.Internal, "failed to list nodes")
	}

	response := grpc_southbound.ListNodesResponse{
		Nodes: make([]*grpc_southbound.NodeResponse, len(nodes)),
	}

	for i, node := range nodes {
		nodeResponse := &grpc_southbound.NodeResponse{
			Node: &grpc_southbound.Node{
				Id:           int32(node.ID),
				SerialNumber: node.SerialNumber,
				NetworkIndex: int32(node.NetworkIndex),
				Locality:     node.Locality,
				VersionSetId: node.VersionSetID.String(),
			},
		}

		if node.LastSeen != nil {
			nodeResponse.Node.LastSeen = timestamppb.New(*node.LastSeen)
		}

		response.Nodes[i] = nodeResponse

		if req.GetInclude() {
			// Get hardware configs
			hwConfigs, err := sb.db.GetHwConfigbyNodeID(ctx, int(node.ID))
			if err != nil {
				log.Warn().Err(err).Msg("failed to get hardware configs")
			} else {
				nodeResponse.HwConfigs = make([]*grpc_southbound.HardwareConfig, len(hwConfigs))
				for i, config := range hwConfigs {
					nodeResponse.HwConfigs[i] = &grpc_southbound.HardwareConfig{
						Id:               int32(config.ID),
						NodeSerialNumber: config.NodeSerial,
						Device:           config.Device,
						IpCidr:           config.IPCIDR,
						VersionSetId:     config.VersionSetID.String(),
					}
				}
			}

			proxies, err := sb.db.GetProxyBySerialNumber(ctx, node.SerialNumber, node.VersionSetID)
			if err != nil {
				log.Warn().Err(err).Msg("failed to get proxies")
			} else {
				nodeResponse.Proxy = make([]*grpc_southbound.Proxy, len(proxies))
				for i, proxy := range proxies {
					nodeResponse.Proxy[i] = &grpc_southbound.Proxy{
						Id:                 int32(proxy.ID),
						Name:               proxy.Name,
						NodeSerialNumber:   proxy.NodeSerial,
						GroupName:          proxy.GroupName,
						State:              proxy.State,
						ProxyType:          grpc_southbound.ProxyType(types.ProxyTypeMap[proxy.ProxyType]),
						ServerEndpointAddr: proxy.ServerEndpointAddr,
						ClientEndpointAddr: proxy.ClientEndpointAddr,
						VersionSetId:       proxy.VersionSetID.String(),
					}
				}
			}
		}
	}

	return &response, nil
}

func (sb *SouthboundService) CreateNode(ctx context.Context, req *grpc_southbound.CreateNodeRequest) (*grpc_southbound.NodeResponse, error) {
	versionSetID, err := uuid.FromString(req.GetVersionSetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	node := &types.Node{
		SerialNumber: req.GetSerialNumber(),
		NetworkIndex: int(req.GetNetworkIndex()),
		Locality:     req.GetLocality(),
		VersionSetID: versionSetID,
		CreatedBy:    req.GetUser(),
	}

	createdNode, err := sb.db.CreateNode(ctx, node)
	if err != nil {
		log.Err(err).Msg("failed to create node")
		return nil, status.Error(codes.Internal, "failed to create node")
	}

	return &grpc_southbound.NodeResponse{
		Node: &grpc_southbound.Node{
			Id:           int32(createdNode.ID),
			SerialNumber: createdNode.SerialNumber,
			NetworkIndex: int32(createdNode.NetworkIndex),
			Locality:     createdNode.Locality,
			VersionSetId: createdNode.VersionSetID.String(),
		},
	}, nil
}

func (sb *SouthboundService) GetNode(ctx context.Context, req *grpc_southbound.GetNodeRequest) (*grpc_southbound.NodeResponse, error) {
	var node *types.Node
	var err error

	switch query := req.GetQuery().(type) {
	case *grpc_southbound.GetNodeRequest_Id:
		node, err = sb.db.GetNodebyID(ctx, int(query.Id))
	case *grpc_southbound.GetNodeRequest_NodeQuery:
		versionSetID, err := uuid.FromString(query.NodeQuery.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
		}
		node, err = sb.db.GetNodebySerial(ctx, query.NodeQuery.GetSerialNumber(), versionSetID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "cannot get node: %v", err)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid query type")
	}

	if err != nil {
		log.Err(err).Msg("failed to get node")
		return nil, status.Error(codes.Internal, "failed to get node")
	}

	response := &grpc_southbound.NodeResponse{
		Node: &grpc_southbound.Node{
			Id:           int32(node.ID),
			SerialNumber: node.SerialNumber,
			NetworkIndex: int32(node.NetworkIndex),
			Locality:     node.Locality,
			VersionSetId: node.VersionSetID.String(),
		},
	}

	if node.LastSeen != nil {
		response.Node.LastSeen = timestamppb.New(*node.LastSeen)
	}

	if req.GetInclude() {
		// Get hardware configs
		hwConfigs, err := sb.db.GetHwConfigbyNodeID(ctx, int(node.ID))
		if err != nil {
			log.Warn().Err(err).Msg("failed to get hardware configs")
		} else {
			response.HwConfigs = make([]*grpc_southbound.HardwareConfig, len(hwConfigs))
			for i, config := range hwConfigs {
				response.HwConfigs[i] = &grpc_southbound.HardwareConfig{
					Id:               int32(config.ID),
					NodeSerialNumber: config.NodeSerial,
					Device:           config.Device,
					IpCidr:           config.IPCIDR,
					VersionSetId:     config.VersionSetID.String(),
				}
			}
		}

		proxies, err := sb.db.GetProxyBySerialNumber(ctx, node.SerialNumber, node.VersionSetID)
		if err != nil {
			log.Warn().Err(err).Msg("failed to get proxies")
		} else {
			response.Proxy = make([]*grpc_southbound.Proxy, len(proxies))
			for i, proxy := range proxies {
				response.Proxy[i] = &grpc_southbound.Proxy{
					Id:                 int32(proxy.ID),
					Name:               proxy.Name,
					NodeSerialNumber:   proxy.NodeSerial,
					GroupName:          proxy.GroupName,
					State:              proxy.State,
					ProxyType:          grpc_southbound.ProxyType(types.ProxyTypeMap[proxy.ProxyType]),
					ServerEndpointAddr: proxy.ServerEndpointAddr,
					ClientEndpointAddr: proxy.ClientEndpointAddr,
					VersionSetId:       proxy.VersionSetID.String(),
				}
			}
		}
	}

	return response, nil
}

func (sb *SouthboundService) UpdateNode(ctx context.Context, req *grpc_southbound.UpdateNodeRequest) (*emptypb.Empty, error) {

	updates := make(map[string]interface{})
	if req.GetNetworkIndex() != 0 {
		updates["network_index"] = req.GetNetworkIndex()
	}
	if req.GetLocality() != "" {
		updates["locality"] = req.GetLocality()
	}
	if req.GetLastSeen() != nil {
		updates["last_seen"] = req.GetLastSeen().AsTime()
	}

	switch query := req.GetQuery().(type) {
	case *grpc_southbound.UpdateNodeRequest_Id:
		where_string := fmt.Sprintf("id = %d", req.GetId())
		err := sb.db.UpdateWhere(ctx, "nodes", updates, where_string)
		if err != nil {
			log.Err(err).Msg("failed to update node")
			return nil, status.Error(codes.Internal, "failed to update node")
		}

	case *grpc_southbound.UpdateNodeRequest_NodeQuery:
		versionSetID, err := uuid.FromString(query.NodeQuery.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
		}
		serialNumber := query.NodeQuery.GetSerialNumber()
		where_string := fmt.Sprintf("serial_number = %s AND version_set_id = %s", serialNumber, versionSetID.String())
		err = sb.db.UpdateWhere(ctx, "nodes", updates, where_string)
		if err != nil {
			log.Err(err).Msg("failed to update node")
			return nil, fmt.Errorf("failed to update node: %w", err)
		}

	default:
		return nil, status.Error(codes.InvalidArgument, "invalid query type")
	}

	return &emptypb.Empty{}, nil
}

func (sb *SouthboundService) DeleteNode(ctx context.Context, req *grpc_southbound.DeleteNodeRequest) (*emptypb.Empty, error) {
	versionSetID, err := uuid.FromString(req.GetVersionSetId())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	err = sb.db.DeleteNode(ctx, req.GetSerialNumber(), versionSetID)
	if err != nil {
		log.Err(err).Msg("failed to delete node")
		return nil, status.Error(codes.Internal, "failed to delete node")
	}

	return &emptypb.Empty{}, nil
}
