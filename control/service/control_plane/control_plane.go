package control_plane

import (
	"github.com/philslol/kritis3m_scalev2/control/types"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
)

type ControlPlane struct {
	v1.UnimplementedControlPlaneServer
	broker *broker
	client *mqtt_client
}

func NewControlPlane(broker_cfg types.BrokerConfig, control_plane_cfg types.ControlPlaneConfig) *ControlPlane {
	broker := new_broker(broker_cfg)

	client := new_mqtt_client(control_plane_cfg)

	return &ControlPlane{
		broker: broker,
		client: client,
	}
}
