package southbound

import (
	"context"
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
		NodeID:             int(req.NodeId),
		GroupID:            int(req.GroupId),
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
		Proxy: &v1.Proxy{
			Id:                 int32(createdProxy.ID),
			NodeId:             int32(createdProxy.NodeID),
			GroupId:            int32(createdProxy.GroupID),
			State:              createdProxy.State,
			ProxyType:          req.ProxyType,
			ServerEndpointAddr: createdProxy.ServerEndpointAddr,
			ClientEndpointAddr: createdProxy.ClientEndpointAddr,
			VersionSetId:       createdProxy.VersionSetID.String(),
			CreatedBy:          createdProxy.CreatedBy,
		},
	}, nil
}

func (sb *SouthboundService) GetProxy(ctx context.Context, req *v1.GetProxyRequest) (*v1.ProxyResponse, error) {
	proxy, err := sb.db.GetProxyByID(ctx, int(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "proxy not found: %v", err)
	}

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

	return &v1.ProxyResponse{
		Proxy: &v1.Proxy{
			Id:                 int32(proxy.ID),
			NodeId:             int32(proxy.NodeID),
			GroupId:            int32(proxy.GroupID),
			State:              proxy.State,
			ProxyType:          proxyType,
			ServerEndpointAddr: proxy.ServerEndpointAddr,
			ClientEndpointAddr: proxy.ClientEndpointAddr,
			VersionSetId:       proxy.VersionSetID.String(),
			CreatedBy:          proxy.CreatedBy,
		},
	}, nil
}

func (sb *SouthboundService) ListProxies(ctx context.Context, req *v1.ListProxiesRequest) (*v1.ListProxiesResponse, error) {
	proxies, err := sb.db.ListProxies(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list proxies: %v", err)
	}

	var proxyResponses []*v1.Proxy
	for _, proxy := range proxies {
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

		proxyResponses = append(proxyResponses, &v1.Proxy{
			Id:                 int32(proxy.ID),
			NodeId:             int32(proxy.NodeID),
			GroupId:            int32(proxy.GroupID),
			State:              proxy.State,
			ProxyType:          proxyType,
			ServerEndpointAddr: proxy.ServerEndpointAddr,
			ClientEndpointAddr: proxy.ClientEndpointAddr,
			VersionSetId:       proxy.VersionSetID.String(),
			CreatedBy:          proxy.CreatedBy,
		})
	}

	return &v1.ListProxiesResponse{
		Proxies: proxyResponses,
	}, nil
}

func (sb *SouthboundService) UpdateProxy(ctx context.Context, req *v1.UpdateProxyRequest) (*empty.Empty, error) {
	updates := make(map[string]interface{})

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
	if req.VersionSetId != nil {
		updates["version_set_id"] = *req.VersionSetId
	}

	err := sb.db.Update(ctx, "proxies", updates, "id", int(req.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update proxy: %v", err)
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
