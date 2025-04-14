package control_plane

import (
	"context"
	"log/slog"
	"os"

	mqtt "github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker"
	auth "github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker/hooks/auth"
	mqtt_listeners "github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker/listeners"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
)

type Broker struct {
	broker_cfg types.BrokerConfig
	log        *zerolog.Logger
	address    string
	id         string

	broker *mqtt.Server
}

func NewBroker(broker_cfg types.BrokerConfig) *Broker {
	capabilities := mqtt.NewDefaultServerCapabilities()
	options := &mqtt.Options{
		Capabilities: capabilities,
	}
	// options.Capabilities = mqtt.NewDefaultServerCapabilities()
	// options.Capabilities.Compatibilities.PassiveClientDisconnect = false
	server := mqtt.New(options)
	err := server.AddHook(new(auth.AllowHook), nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Error adding auth hook")
	}
	// convert log level to slog level
	var log_level slog.Level
	if broker_cfg.Log.Level == zerolog.DebugLevel {
		log_level = slog.LevelDebug
	} else if broker_cfg.Log.Level == zerolog.InfoLevel {
		log_level = slog.LevelInfo
	} else if broker_cfg.Log.Level == zerolog.WarnLevel {
		log_level = slog.LevelWarn
	} else if broker_cfg.Log.Level == zerolog.ErrorLevel {
		log_level = slog.LevelError
	} else {
		log_level = slog.LevelInfo
	}

	server.Log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: log_level,
	}))
	//create component zerologger
	zerologger := zerolog.New(os.Stdout).Level(zerolog.Level(broker_cfg.Log.Level))

	return &Broker{
		broker_cfg: broker_cfg,
		log:        &zerologger,
		broker:     server,
		address:    broker_cfg.Adress,
		id:         "broker",
	}
}

func (b *Broker) Serve(ctx context.Context) error {
	var listener mqtt_listeners.Listener
	var err error

	// Select appropriate listener based on configuration
	if b.broker_cfg.TcpOnly {
		listener = mqtt_listeners.NewTCP(mqtt_listeners.Config{
			Type:    mqtt_listeners.TypeTCP,
			ID:      b.id,
			Address: b.address,
		})
	} else {
		listener = mqtt_listeners.NewASLListener(
			mqtt_listeners.Config{
				ID:      b.id,
				Address: b.address,
			},
			b.broker_cfg.EndpointConfig,
		)
	}

	// Add listener to broker
	if err = b.broker.AddListener(listener); err != nil {
		b.log.Err(err).Msg("Cannot add listener to broker")
		return err
	}

	// Start the broker in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := b.broker.Serve(); err != nil {
			b.log.Err(err).Msg("Broker serve failed")
			errChan <- err
		}
	}()

	// Wait for context cancellation or an error from broker
	select {
	case <-ctx.Done():
		b.log.Info().Msg("Broker shutdown signal received")
		if err := ctx.Err(); err != nil {
			b.log.Info().Err(err).Msg("Context cancellation reason")
		}
	case err := <-errChan:
		b.log.Err(err).Msg("Broker encountered an error")
	}

	b.broker.Close()
	b.log.Info().Msg("Broker stopped")
	return nil
}
