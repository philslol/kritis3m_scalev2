package control

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/service/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"

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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Create EST server with the application context
	estServer, err := controlplane.NewESTServer(&scale.cfg.ESTServer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create EST server")
	} else {
		// Replace the EST server's context with the application context
		estServer.SetContext(ctx)

		// Start the EST server in a goroutine
		go func() {
			log.Info().Msg("Starting EST server")
			if err := estServer.Serve(); err != nil {
				log.Error().Err(err).Msg("EST server serve failed")
				cancel() // Cancel context on EST server failure
			}
			log.Info().Msg("EST server stopped")
			log.Info().Msg("EST server stopped\n\n")
		}()

		// Ensure server is properly shut down
		defer estServer.Shutdown()
	}

	database, err := db.NewStateManager(ctx)
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

	sb := southbound.NewSouthbound(database, scale.cfg.CliConfig.ServerAddr)

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

	go func() {
		log.Info().Msgf("Server listening at %v", lis.Addr())
		if err := s.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("gRPC server error")
			cancel() // Cancel context on failure
		}
	}()
	time.Sleep(1 * time.Second)

	hello_service := southbound.NewHelloService(database, scale.cfg.CliConfig.ServerAddr)
	go func() {
		err := hello_service.Hello(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Hello service error")
			cancel() // Cancel context on failure
		}
	}()

	log_service := southbound.NewLogService(database, scale.cfg.CliConfig.ServerAddr, scale.cfg.Logfile)
	go func() {
		err := log_service.LogNodeTransaction(ctx)
		if err != nil {
			log.Fatal().Err(err).Msg("Log service error")
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

// small helper, should not be used in production, but is useful for testing
// no grpc middleware required to execute db operations
func (scale *Kritis3m_Scale) GetRawDB(timeout time.Duration) (*db.StateManager, context.Context, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	database, err := db.NewStateManager(ctx)
	if err != nil {
		log.Err(err).Msg("Failed to get raw database")
		return nil, ctx, cancel, err
	}
	return database, ctx, cancel, nil
}
