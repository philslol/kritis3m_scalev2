package southbound

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	empty "github.com/golang/protobuf/ptypes/empty"
	"github.com/rs/zerolog/log"

	"github.com/gofrs/uuid/v5"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (sb *SouthboundService) CreateProxy(ctx context.Context, req *v1.CreateProxyRequest) (*v1.ProxyResponse, error) {
	versionSetUUID, err := uuid.FromString(req.VersionSetId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
	}

	// Convert ProxyType enum to lowercase string
	proxyTypeStr := strings.ToLower(req.ProxyType.String())
	log.Debug().Msgf("proxyTypeStr: %s", proxyTypeStr)

	proxy := &types.Proxy{
		Name:               req.Name,
		NodeSerial:         req.NodeSerialNumber,
		GroupName:          req.GroupName,
		State:              req.State,
		ProxyType:          types.ProxyType(proxyTypeStr),
		ServerEndpointAddr: req.ServerEndpointAddr,
		ClientEndpointAddr: req.ClientEndpointAddr,
		VersionSetID:       versionSetUUID,
		CreatedBy:          req.CreatedBy,
	}

	createdProxy, err := sb.db.CreateProxy(ctx, proxy)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create proxy: %v", err)
	}

	return &v1.ProxyResponse{
		Proxy: []*v1.Proxy{
			{
				Id:                 int32(createdProxy.ID),
				NodeSerialNumber:   createdProxy.NodeSerial,
				GroupName:          createdProxy.GroupName,
				State:              createdProxy.State,
				ProxyType:          req.ProxyType,
				ServerEndpointAddr: createdProxy.ServerEndpointAddr,
				ClientEndpointAddr: createdProxy.ClientEndpointAddr,
				VersionSetId:       createdProxy.VersionSetID.String(),
				Name:               createdProxy.Name,
				CreatedBy:          createdProxy.CreatedBy,
			},
		},
	}, nil
}

func (sb *SouthboundService) GetProxy(ctx context.Context, req *v1.GetProxyRequest) (*v1.ProxyResponse, error) {
	var proxies []*types.Proxy
	switch query := req.GetQuery().(type) {
	case *v1.GetProxyRequest_Id:
		proxy, err := sb.db.GetProxyByID(ctx, int(req.GetId()))
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "proxy not found: %v", err)
		}
		proxies = append(proxies, proxy)
	case *v1.GetProxyRequest_NameQuery:
		versionSetID, err := uuid.FromString(query.NameQuery.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "uuid conversion failed: %v", err)
		}
		proxy_name := query.NameQuery.GetName()
		proxy, err := sb.db.GetProxyByName(ctx, proxy_name, versionSetID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "proxy not found: %v", err)
		}
		proxies = append(proxies, proxy)
	case *v1.GetProxyRequest_SerialQuery:
		versionSetID, err := uuid.FromString(query.SerialQuery.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "uuid conversion failed: %v", err)
		}
		serialNumber := query.SerialQuery.GetSerial()
		proxies, err = sb.db.GetProxyBySerialNumber(ctx, serialNumber, versionSetID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "proxy not found: %v", err)
		}
	case *v1.GetProxyRequest_VersionSetId:
		versionSetID, err := uuid.FromString(query.VersionSetId)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid version set ID: %v", err)
		}
		proxies, err = sb.db.GetProxyByVersionSetID(ctx, versionSetID)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "proxy not found: %v", err)
		}

	default:
		var err error
		proxies, err = sb.db.GetAllProxies(ctx)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, "proxy not found: %v", err)
		}
	}
	//convert proxies and return
	proxyResponses := make([]*v1.Proxy, len(proxies))
	for i, proxy := range proxies {

		// Convert stored string back to ProxyType enum
		proxyType := v1.ProxyType_FORWARD // Default to FORWARD
		switch strings.ToUpper(string(proxy.ProxyType)) {
		case "FORWARD":
			proxyType = v1.ProxyType_FORWARD
		case "REVERSE":
			proxyType = v1.ProxyType_REVERSE
		case "TLSTLS":
			proxyType = v1.ProxyType_TLSTLS
		}

		proxyResponses[i] = &v1.Proxy{
			Id:                 int32(proxy.ID),
			NodeSerialNumber:   proxy.NodeSerial,
			GroupName:          proxy.GroupName,
			State:              proxy.State,
			ProxyType:          proxyType,
			ServerEndpointAddr: proxy.ServerEndpointAddr,
			ClientEndpointAddr: proxy.ClientEndpointAddr,
			VersionSetId:       proxy.VersionSetID.String(),
		}
	}

	return &v1.ProxyResponse{
		Proxy: proxyResponses,
	}, nil
}

func (sb *SouthboundService) UpdateProxy(ctx context.Context, req *v1.UpdateProxyRequest) (*empty.Empty, error) {
	updates := make(map[string]interface{})
	var err error

	if req.State != nil {
		updates["state"] = *req.State
	}

	if req.ProxyType != nil {
		updates["proxy_type"] = strings.ToLower(req.ProxyType.String())
	}
	if req.ServerEndpointAddr != nil {
		updates["server_endpoint_addr"] = *req.ServerEndpointAddr
	}
	if req.ClientEndpointAddr != nil {
		updates["client_endpoint_addr"] = *req.ClientEndpointAddr
	}
	if req.GroupId != nil {
		updates["group_id"] = *req.GroupId
	}

	switch query := req.GetQuery().(type) {
	case *v1.UpdateProxyRequest_Id:
		where_string := fmt.Sprintf("id = %d", query.Id)
		err = sb.db.UpdateWhere(ctx, "proxies", updates, where_string)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update proxy: %v", err)
		}
	case *v1.UpdateProxyRequest_NameQuery:
		versionSetID, err := uuid.FromString(query.NameQuery.GetVersionSetId())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "uuid conversion failed: %v", err)
		}
		proxy_name := query.NameQuery.GetName()
		where_string := fmt.Sprintf("name = %s AND version_set_id = %s", proxy_name, versionSetID.String())
		err = sb.db.UpdateWhere(ctx, "proxies", updates, where_string)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to update proxy: %v", err)
		}
	}

	return &empty.Empty{}, nil
}

func (sb *SouthboundService) DeleteProxy(ctx context.Context, req *v1.DeleteProxyRequest) (*empty.Empty, error) {
	err := sb.db.Delete(ctx, "proxies", "id", strconv.Itoa(int(req.Id)))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete proxy: %v", err)
	}

	return &empty.Empty{}, nil
}
