package southbound

import (
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	//include db
	"context"

	empty "github.com/golang/protobuf/ptypes/empty"
	db "github.com/philslol/kritis3m_scalev2/control/db"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

//It doesnt matter if cli or ui uses southbound service

type SouthboundService struct {
	db *db.StateManager
	v1.UnimplementedSouthboundServer
}

func NewSouthbound(db *db.StateManager) *SouthboundService {
	return &SouthboundService{
		db: db,
	}
}

func (sb *SouthboundService) CreateGroup(ctx context.Context, req *v1.CreateGroupRequest) (*v1.GroupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateGroup not implemented")
}
func (sb *SouthboundService) GetGroup(ctx context.Context, req *v1.GetGroupRequest) (*v1.GroupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetGroup not implemented")
}
func (sb *SouthboundService) ListGroups(ctx context.Context, req *v1.ListGroupsRequest) (*v1.ListGroupsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListGroups not implemented")
}
func (sb *SouthboundService) UpdateGroup(ctx context.Context, req *v1.UpdateGroupRequest) (*empty.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateGroup not implemented")
}
func (sb *SouthboundService) DeleteGroup(ctx context.Context, req *v1.DeleteGroupRequest) (*empty.Empty, error) {
	return &empty.Empty{}, status.Errorf(codes.Unimplemented, "method DeleteGroup not implemented")
}
