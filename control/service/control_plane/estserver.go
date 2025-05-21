package control_plane

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	aslListener "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl/listener"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/aslhttpserver"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/common"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/est"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/kritis3m_pki"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/realca"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ZerologAdapter wraps zerolog.Logger to implement common.Logger interface
type ZerologAdapter struct {
	logger zerolog.Logger
}

// NewZerologAdapter creates a new ZerologAdapter
func NewZerologAdapter(logger zerolog.Logger) *ZerologAdapter {
	return &ZerologAdapter{logger: logger}
}

// Errorf implements common.Logger
func (z *ZerologAdapter) Errorf(format string, args ...interface{}) {
	z.logger.Error().Msgf(format, args...)
}

// Errorw implements common.Logger
func (z *ZerologAdapter) Errorw(format string, keysAndValues ...interface{}) {
	event := z.logger.Error()
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, val := keysAndValues[i], keysAndValues[i+1]
			if k, ok := key.(string); ok {
				event = event.Interface(k, val)
			}
		}
	}
	event.Msg(format)
}

// Infof implements common.Logger
func (z *ZerologAdapter) Infof(format string, args ...interface{}) {
	z.logger.Info().Msgf(format, args...)
}

// Infow implements common.Logger
func (z *ZerologAdapter) Infow(format string, keysAndValues ...interface{}) {
	event := z.logger.Info()
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, val := keysAndValues[i], keysAndValues[i+1]
			if k, ok := key.(string); ok {
				event = event.Interface(k, val)
			}
		}
	}
	event.Msg(format)
}

// Debugf implements common.Logger
func (z *ZerologAdapter) Debugf(format string, args ...interface{}) {
	z.logger.Debug().Msgf(format, args...)
}

// Debugw implements common.Logger
func (z *ZerologAdapter) Debugw(format string, keysAndValues ...interface{}) {
	event := z.logger.Debug()
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, val := keysAndValues[i], keysAndValues[i+1]
			if k, ok := key.(string); ok {
				event = event.Interface(k, val)
			}
		}
	}
	event.Msg(format)
}

// With implements common.Logger
func (z *ZerologAdapter) With(keysAndValues ...interface{}) common.Logger {
	ctx := z.logger.With()
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, val := keysAndValues[i], keysAndValues[i+1]
			if k, ok := key.(string); ok {
				ctx = ctx.Interface(k, val)
			}
		}
	}
	return &ZerologAdapter{logger: ctx.Logger()}
}

// ESTServer represents the EST server instance
type ESTServer struct {
	server   *aslhttpserver.ASLServer
	endpoint *asl.ASLEndpoint
}

// NewESTServer creates and sets up a new EST server based on the provided configuration
func NewESTServer(cfg *types.ESTServerConfig) (*ESTServer, error) {
	var err error

	err = kritis3m_pki.InitPKI(&kritis3m_pki.KRITIS3MPKIConfiguration{
		LoggingEnabled: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PKI: %v", err)
	}

	zLogger := types.CreateLogger("estserver", cfg.Log.Level, cfg.Log.File)
	// Create an adapter that implements common.Logger interface
	logger := NewZerologAdapter(zLogger)

	// Set default validity if not specified
	validity := cfg.CA.Validity
	if validity == 0 {
		logger.Infof("No validity specified in configuration file, using default value of 365 days")
		validity = 365
	}

	// Create CA using the default backend
	if cfg.CA.DefaultBackend == nil {
		return nil, fmt.Errorf("no default backend configured")
	}

	ca, err := realca.New(cfg.CA.Backends, cfg.CA.DefaultBackend, logger, validity)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA: %w", err)
	}

	// Create server router
	r, err := est.NewRouter(&est.ServerConfig{
		CA:           ca,
		Logger:       logger,
		AllowedHosts: cfg.AllowedHosts,
		Timeout:      time.Duration(cfg.Timeout) * time.Second,
		RateLimit:    cfg.RateLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create new EST router: %v", err)
	}

	endpoint := asl.ASLsetupServerEndpoint(&cfg.EndpointConfig)
	if endpoint == nil {
		return nil, fmt.Errorf("failed to setup server endpoint")

	}

	// Create ASL HTTP server
	aslServer := &aslhttpserver.ASLServer{
		Server: &http.Server{
			Addr:    cfg.ServerAddress,
			Handler: r,
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				if aslConn, ok := c.(*aslListener.ASLConn); ok {
					if aslConn.TLSState != nil {
						return context.WithValue(ctx, common.TLSStateKey, aslConn.TLSState)
					}
				}
				return ctx
			},
		},
		ASLTLSEndpoint: endpoint,
		Logger:         logger,
		DebugLog:       false,
	}

	return &ESTServer{
		server:   aslServer,
		endpoint: endpoint,
	}, nil
}

// Serve starts the EST server
func (e *ESTServer) Serve(ctx context.Context) error {
	errChan := make(chan error)
	go func() {
		err := e.server.ListenAndServeASLTLS()
		if err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("EST server error")
			errChan <- err
		}
		close(errChan)
	}()
	select {
	case <-ctx.Done():
		log.Info().Msg("EST server context done")
		return ctx.Err()
	case err := <-errChan:
		log.Info().Msg("EST server error")
		return err
	}
}

// shutdownInternal handles the internal shutdown logic
func (e *ESTServer) Shutdown() {
	if e.server != nil && e.server.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := e.server.Server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Error shutting down EST server")
		}
	}

	if e.endpoint != nil {
		asl.ASLFreeEndpoint(e.endpoint)
	}

}
