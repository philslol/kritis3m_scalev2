package southbound

import (
	"github.com/philslol/kritis3m_scalev2/control/db"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

// SouthboundService handles communication with the control plane
type SouthboundService struct {
	db   *db.StateManager
	addr string
	v1.UnimplementedSouthboundServer
}

// NewSouthbound creates a new instance of SouthboundService
func NewSouthbound(db *db.StateManager, addr string) *SouthboundService {
	//create new controlplane client
	return &SouthboundService{
		db:   db,
		addr: addr,
	}
}
