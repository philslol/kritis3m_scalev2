package controlplane

import (
	"log/slog"
	"os"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	mqtt "github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker/listeners"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
)

type broker struct {
	ep      asl.EndpointConfig
	log     *zerolog.Logger
	address string
	id      string

	broker *mqtt.Server
}

func new_broker(broker_cfg types.BrokerConfig) *broker {
	options := &mqtt.Options{}
	// options.Capabilities = mqtt.NewDefaultServerCapabilities()
	// options.Capabilities.Compatibilities.PassiveClientDisconnect = false

	server := mqtt.New(options)
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

	return &broker{
		ep:      broker_cfg.EndpointConfig,
		log:     &zerologger,
		broker:  server,
		address: broker_cfg.Adress,
		id:      "broker",
	}
}

func (b *broker) serve() error {
	//create listner
	var err error

	asl := listeners.NewASLListener(
		listeners.Config{
			ID:      b.id,
			Address: b.address,
		},
		b.ep,
	)

	err = b.broker.AddListener(asl)
	if err != nil {
		b.log.Err(err).Msg("Cant serve broker")
		return err
	}

	go func() {
		err := b.broker.Serve()
		if err != nil {
			b.log.Fatal().Err(err).Msg("Cant serve broker")
		}
	}()
	select {}

	return nil
}
