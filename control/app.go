package control

import (
	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/service/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"net"

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

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal().Err(err)
	}
	s := grpc.NewServer()

	v1.RegisterSouthboundServer(s, sb)

	log.Printf("server listening at %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		// log.Fatalf("failed to serve: %v", err)
	}

	return
}
