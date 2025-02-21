package mock

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
)

// MockSouthboundClient is a mock implementation of the SouthboundClient interface
type MockSouthboundClient struct {
	mock.Mock
}

// Node operations
func (m *MockSouthboundClient) CreateNode(ctx context.Context, req *v1.CreateNodeRequest, opts ...grpc.CallOption) (*v1.NodeResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.NodeResponse), args.Error(1)
}

func (m *MockSouthboundClient) GetNode(ctx context.Context, req *v1.GetNodeRequest, opts ...grpc.CallOption) (*v1.NodeResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.NodeResponse), args.Error(1)
}

func (m *MockSouthboundClient) ListNodes(ctx context.Context, req *v1.ListNodesRequest, opts ...grpc.CallOption) (*v1.ListNodesResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.ListNodesResponse), args.Error(1)
}

func (m *MockSouthboundClient) UpdateNode(ctx context.Context, req *v1.UpdateNodeRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

func (m *MockSouthboundClient) DeleteNode(ctx context.Context, req *v1.DeleteNodeRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

// EndpointConfig operations
func (m *MockSouthboundClient) CreateEndpointConfig(ctx context.Context, req *v1.CreateEndpointConfigRequest, opts ...grpc.CallOption) (*v1.EndpointConfig, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.EndpointConfig), args.Error(1)
}

func (m *MockSouthboundClient) GetEndpointConfig(ctx context.Context, req *v1.GetEndpointConfigRequest, opts ...grpc.CallOption) (*v1.EndpointConfig, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.EndpointConfig), args.Error(1)
}

func (m *MockSouthboundClient) ListEndpointConfigs(ctx context.Context, req *v1.ListEndpointConfigsRequest, opts ...grpc.CallOption) (*v1.ListEndpointConfigsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.ListEndpointConfigsResponse), args.Error(1)
}

func (m *MockSouthboundClient) UpdateEndpointConfig(ctx context.Context, req *v1.UpdateEndpointConfigRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

func (m *MockSouthboundClient) DeleteEndpointConfig(ctx context.Context, req *v1.DeleteEndpointConfigRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

// Group operations
func (m *MockSouthboundClient) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest, opts ...grpc.CallOption) (*v1.GroupResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.GroupResponse), args.Error(1)
}

func (m *MockSouthboundClient) GetGroup(ctx context.Context, req *v1.GetGroupRequest, opts ...grpc.CallOption) (*v1.GroupResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.GroupResponse), args.Error(1)
}

func (m *MockSouthboundClient) ListGroups(ctx context.Context, req *v1.ListGroupsRequest, opts ...grpc.CallOption) (*v1.ListGroupsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.ListGroupsResponse), args.Error(1)
}

func (m *MockSouthboundClient) UpdateGroup(ctx context.Context, req *v1.UpdateGroupRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

func (m *MockSouthboundClient) DeleteGroup(ctx context.Context, req *v1.DeleteGroupRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

// HardwareConfig operations
func (m *MockSouthboundClient) CreateHardwareConfig(ctx context.Context, req *v1.CreateHardwareConfigRequest, opts ...grpc.CallOption) (*v1.HardwareConfigResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.HardwareConfigResponse), args.Error(1)
}

func (m *MockSouthboundClient) GetHardwareConfig(ctx context.Context, req *v1.GetHardwareConfigRequest, opts ...grpc.CallOption) (*v1.HardwareConfigResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.HardwareConfigResponse), args.Error(1)
}

func (m *MockSouthboundClient) ListHardwareConfigs(ctx context.Context, req *v1.ListHardwareConfigsRequest, opts ...grpc.CallOption) (*v1.ListHardwareConfigsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.ListHardwareConfigsResponse), args.Error(1)
}

func (m *MockSouthboundClient) UpdateHardwareConfig(ctx context.Context, req *v1.UpdateHardwareConfigRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

func (m *MockSouthboundClient) DeleteHardwareConfig(ctx context.Context, req *v1.DeleteHardwareConfigRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

// Version Set operations
func (m *MockSouthboundClient) CreateVersionSet(ctx context.Context, req *v1.CreateVersionSetRequest, opts ...grpc.CallOption) (*v1.VersionSetResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.VersionSetResponse), args.Error(1)
}

func (m *MockSouthboundClient) GetVersionSet(ctx context.Context, req *v1.GetVersionSetRequest, opts ...grpc.CallOption) (*v1.VersionSetResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.VersionSetResponse), args.Error(1)
}

func (m *MockSouthboundClient) ListVersionSets(ctx context.Context, req *v1.ListVersionSetsRequest, opts ...grpc.CallOption) (*v1.ListVersionSetsResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.ListVersionSetsResponse), args.Error(1)
}

func (m *MockSouthboundClient) UpdateVersionSet(ctx context.Context, req *v1.UpdateVersionSetRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

func (m *MockSouthboundClient) DeleteVersionSet(ctx context.Context, req *v1.DeleteVersionSetRequest, opts ...grpc.CallOption) (*empty.Empty, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*empty.Empty), args.Error(1)
}

func (m *MockSouthboundClient) ActivateVersionSet(ctx context.Context, req *v1.ActivateVersionSetRequest, opts ...grpc.CallOption) (*v1.VersionSetResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.VersionSetResponse), args.Error(1)
}

func (m *MockSouthboundClient) DisableVersionSet(ctx context.Context, req *v1.DisableVersionSetRequest, opts ...grpc.CallOption) (*v1.VersionSetResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*v1.VersionSetResponse), args.Error(1)
}
