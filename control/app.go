package control

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	grpc_control_plane "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/control_plane"
	grpc_est "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/est"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/service/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"

	controlplane "github.com/philslol/kritis3m_scalev2/control/service/control_plane"
)

func init() {
	// Set global log level to debug
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

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
	log.Info().Msgf("Entrypoint function serve")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	var config asl.ASLConfig = asl.ASLConfig{
		LogLevel: scale.cfg.ASLConfig.LogLevel,
	}
	asl.ASLinit(&config)

	estServer, err := controlplane.NewESTServer(&scale.cfg.ESTServer)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	} else {
		go func() {
			if err := estServer.Serve(ctx); err != nil {
				log.Error().Err(err).Msg("EST server serve failed")
				ctx.Done()
			}
		}()

		// Ensure server is properly shut down
		defer estServer.Shutdown()
	}

	database, err := db.NewStateManager(ctx, scale.cfg.Log)
	if err != nil {
		log.Err(err).Msg("")
	}

	broker := controlplane.NewBroker(scale.cfg.Broker)
	if broker == nil {
		log.Err(err).Msg("Broker is nil")
	}
	broker_ctx, broker_cancel := context.WithCancel(context.Background())
	defer broker_cancel()

	go func() {
		if err := broker.Serve(broker_ctx); err != nil {
			log.Err(err).Msg("Broker serve failed")
		}
	}()

	control_plane := controlplane.ControlPlaneInit(scale.cfg.ControlPlane)
	if control_plane == nil {
		log.Err(err).Msg("Control Plane is nil")
	}

	sb := southbound.NewSouthbound(database, scale.cfg.CliConfig.ServerAddr)
	lis, err := net.Listen("tcp", scale.cfg.CliConfig.ServerAddr)
	if err != nil {
		log.Err(err).Msg("")
	}

	s := grpc.NewServer()
	if err != nil {
		log.Err(err).Msg("")
	}

	grpc_southbound.RegisterSouthboundServer(s, sb)
	grpc_est.RegisterEstServiceServer(s, sb)
	grpc_control_plane.RegisterControlPlaneServer(s, control_plane)

	go func() {
		log.Info().Msgf("Server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Err(err).Msg("gRPC server error")
			ctx.Done()
		}
	}()

	hello_service := southbound.NewHelloService(database, scale.cfg.CliConfig.ServerAddr, scale.cfg.Log)
	go func() {
		err := hello_service.Hello(ctx)
		if err != nil {
			log.Err(err).Msg("Hello service error")
			ctx.Done()
		}
	}()

	log_service := southbound.NewLogService(database, scale.cfg.CliConfig.ServerAddr, scale.cfg.Log)
	go func() {
		err := log_service.LogNodeTransaction(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Log service error")
			ctx.Done()
		}
	}()

	// Wait for termination signal
	select {
	case <-signalChan:
		log.Info().Msg("Shutdown signal received")
	case <-ctx.Done():
		log.Info().Msg("Context cancelled")
	case <-broker_ctx.Done():
		log.Info().Msg("Broker context cancelled")
	}
	log.Info().Msg("shutting down")

	// Graceful shutdown
	s.GracefulStop()
	log.Info().Msg("gRPC server stopped")

	log.Info().Msg("Shutdown complete")
}

// small helper, should not be used in production, but is useful for testing
// no grpc middleware required to execute db operations
func (scale *Kritis3m_Scale) GetRawDB(timeout time.Duration) (*db.StateManager, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	database, err := db.NewStateManager(ctx, scale.cfg.Log)
	if err != nil {
		log.Err(err).Msg("Failed to get raw database")
		return nil, ctx, cancel, err
	}
	return database, ctx, cancel, nil
}
