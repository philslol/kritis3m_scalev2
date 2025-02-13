package control

import (

	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/types"

	"github.com/rs/zerolog/log"
)

type Kritis3m_Scale struct {
	cfg *types.Config
}

func NewKritis3m_scale(cfg *types.Config) (*Kritis3m_Scale, error) {
	log.Info().Msg("In function new kritis3m_scale")

	app := Kritis3m_Scale{
		cfg: cfg,
		// noisePrivateKey: noisePrivateKey,
	}

	return &app, nil
}

func (scale *Kritis3m_Scale) Serve() {
	_, err := db.NewStateManager()
	if err != nil{
		log.Err(err)
	}

	return
}
