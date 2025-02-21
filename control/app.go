package control

import (
	"net"

	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/service/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
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
	database, err := db.NewStateManager()
	if err != nil {
		log.Err(err)
	}
	sb := southbound.NewSouthbound(database)

	//use ServerAddr and create new grpc listening server
	lis, err := net.Listen("tcp", scale.cfg.CliConfig.ServerAddr)
	if err != nil {
		log.Fatal().Err(err)
	}

	s := grpc.NewServer()
	if err != nil {
		log.Fatal().Err(err)
	}

	v1.RegisterSouthboundServer(s, sb)
	err = s.Serve(lis)
	if err != nil {
		log.Fatal().Err(err)
	}

	log.Printf("server listening at %v", lis.Addr())

	return
}
