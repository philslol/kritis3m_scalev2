package southbound

import (
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	//include db

	db "github.com/philslol/kritis3m_scalev2/control/db"
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
