package control

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/service/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	controlplane "github.com/philslol/kritis3m_scalev2/control/service/control_plane"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	database, err := db.NewStateManager()
	if err != nil {
		log.Err(err)
	}

	broker := controlplane.NewBroker(scale.cfg.Broker)
	if broker == nil {
		log.Err(err).Msg("Broker is nil")
	}

	go func() {
		if err := broker.Serve(ctx); err != nil {
			log.Err(err).Msg("Broker serve failed")
			cancel() // Cancel context on broker failure
		}
	}()

	control_plane := controlplane.ControlPlaneInit(scale.cfg.ControlPlane)
	if control_plane == nil {
		log.Err(err).Msg("Control Plane is nil")
		log.Fatal().Msg("Control Plane is nil")
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
	v1.RegisterControlPlaneServer(s, control_plane)

	// Start gRPC server in a goroutine
	go func() {
		log.Info().Msgf("Server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("gRPC server error")
			cancel() // Cancel context on failure
		}
	}()

	// Wait for termination signal
	select {
	case <-signalChan:
		log.Info().Msg("Shutdown signal received")
	case <-ctx.Done():
		log.Info().Msg("Context cancelled")
	}

	// Graceful shutdown
	s.GracefulStop()
	log.Info().Msg("gRPC server stopped")

	cancel() // Ensure all goroutines stop
	log.Info().Msg("Shutdown complete")
}
