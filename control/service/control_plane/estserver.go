package control_plane

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	aslListener "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl/listener"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/alogger"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/aslhttpserver"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/common"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/est"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/kritis3m_pki"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/realca"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
	zerolog_log "github.com/rs/zerolog/log"
)

var estLog zerolog.Logger

// ESTServer represents the EST server instance
type ESTServer struct {
	server   *aslhttpserver.ASLServer
	endpoint *asl.ASLEndpoint
	logFile  *os.File
}

// NewESTServer creates and sets up a new EST server based on the provided configuration
func NewESTServer(cfg *types.ESTServerConfig) (*ESTServer, error) {
	var err error

	err = kritis3m_pki.InitPKI(&kritis3m_pki.KRITIS3MPKIConfiguration{
		LogLevel:       int32(cfg.ASLConfig.LogLevel),
		LoggingEnabled: cfg.ASLConfig.LoggingEnabled,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PKI: %v", err)
	}

	var logger est.Logger
	var logFile *os.File

	estLogLevel := cfg.Log.Level
	estLog = zerolog_log.Logger.Level(cfg.Log.Level)
	if cfg.Log.Format == "" {
		logger = alogger.New(os.Stderr, 4)
	} else {
		logFilePath := "/tmp/estserver.log"
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %v", err)
		}

		logger = alogger.New(f, estLogLevel)
		logFile = f
	}

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
		if logFile != nil {
			logFile.Close()
		}
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
		if logFile != nil {
			logFile.Close()
		}
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
		DebugLog:       cfg.Log.Level == zerolog.DebugLevel,
	}

	return &ESTServer{
		server:   aslServer,
		endpoint: endpoint,
		logFile:  logFile,
	}, nil
}

// Serve starts the EST server
func (e *ESTServer) Serve(ctx context.Context) error {
	errChan := make(chan error)
	go func() {
		err := e.server.ListenAndServeASLTLS()
		if err != nil && err != http.ErrServerClosed {
			estLog.Error().Err(err).Msg("EST server error")
			errChan <- err
		}
		close(errChan)
	}()
	select {
	case <-ctx.Done():
		estLog.Info().Msg("EST server context done")
		return ctx.Err()
	case err := <-errChan:
		estLog.Info().Msg("EST server error")
		return err
	}

}

// shutdownInternal handles the internal shutdown logic
func (e *ESTServer) Shutdown() {
	if e.server != nil && e.server.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := e.server.Server.Shutdown(ctx); err != nil {
			estLog.Error().Err(err).Msg("Error shutting down EST server")
		}
	}

	if e.endpoint != nil {
		asl.ASLFreeEndpoint(e.endpoint)
	}

	if e.logFile != nil {
		e.logFile.Close()
	}
}
