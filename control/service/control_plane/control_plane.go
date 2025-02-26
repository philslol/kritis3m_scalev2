package control_plane

import (
	"github.com/philslol/kritis3m_scalev2/control/types"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

type ControlPlane struct {
	v1.UnimplementedControlPlaneServer
	broker *broker
}

func NewControlPlane(broker_cfg types.BrokerConfig, control_plane_cfg types.ControlPlaneConfig) *ControlPlane {
	broker := new_broker(broker_cfg)

	return &ControlPlane{
		broker: broker,
	}
}

func (cp *ControlPlane) Init() error {
	return cp.broker.serve()
}
