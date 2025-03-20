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

// ESTServer represents the EST server instance
type ESTServer struct {
	server     *aslhttpserver.ASLServer
	endpoint   *asl.ASLEndpoint
	logFile    *os.File
	cancelFunc context.CancelFunc
	ctx        context.Context
}

// SetContext replaces the server's context with a new one
func (e *ESTServer) SetContext(ctx context.Context) {
	// Cancel the original context
	if e.cancelFunc != nil {
		e.cancelFunc()
	}

	// Create new derived context
	derivedCtx, cancel := context.WithCancel(ctx)
	e.ctx = derivedCtx
	e.cancelFunc = cancel

	// Monitor parent context
	go func() {
		<-ctx.Done()
		zerolog_log.Debug().Msg("Parent context done, cancelling EST server context")
		cancel()
	}()
}

// NewESTServer creates and sets up a new EST server based on the provided configuration
func NewESTServer(cfg *types.ESTServerConfig) (*ESTServer, error) {
	var err error
	ctx, cancel := context.WithCancel(context.Background())

	// PKCS11 configuration for the CA
	pkcs11Config := kritis3m_pki.PKCS11Config{
		EntityModule: nil,
		IssuerModule: &kritis3m_pki.PKCS11Module{
			Path: cfg.CA.PKCS11Module.Path,
			Pin:  cfg.CA.PKCS11Module.Pin,
			Slot: 0, // TODO: Add slot to configuration
		},
	}

	// Setup ASL endpoint configuration
	endpointConfig := &asl.EndpointConfig{
		MutualAuthentication: cfg.TLS.ASLEndpoint.MutualAuthentication,
		ASLKeyExchangeMethod: asl.KEX_DEFAULT, // TODO: Add key exchange method to configuration
		Ciphersuites:         cfg.TLS.ASLEndpoint.Ciphersuites,
		PreSharedKey: asl.PreSharedKey{
			Enable: false,
		},
		DeviceCertificateChain: asl.DeviceCertificateChain{Path: cfg.TLS.Certificates},
		PrivateKey: asl.PrivateKey{
			Path: cfg.TLS.PrivateKey,
		},
		RootCertificates: asl.RootCertificates{Paths: cfg.TLS.ClientCAs},
		KeylogFile:       cfg.TLS.ASLEndpoint.KeylogFile,
		PKCS11: asl.PKCS11ASL{
			Path: cfg.TLS.PKCS11Module.Path,
			Pin:  cfg.TLS.PKCS11Module.Pin,
		},
	}

	// Create ASL endpoint
	endpoint := asl.ASLsetupServerEndpoint(endpointConfig)
	if endpoint == nil {
		cancel()
		return nil, fmt.Errorf("failed to setup server endpoint")
	}

	// Initialize PKI
	pkiLogLevel := kritis3m_pki.KRITIS3M_PKI_LOG_LEVEL_WRN
	switch cfg.Log.Level {
	case zerolog.ErrorLevel:
		pkiLogLevel = kritis3m_pki.KRITIS3M_PKI_LOG_LEVEL_ERR
	case zerolog.WarnLevel:
		pkiLogLevel = kritis3m_pki.KRITIS3M_PKI_LOG_LEVEL_WRN
	case zerolog.InfoLevel:
		pkiLogLevel = kritis3m_pki.KRITIS3M_PKI_LOG_LEVEL_INF
	case zerolog.DebugLevel:
		pkiLogLevel = kritis3m_pki.KRITIS3M_PKI_LOG_LEVEL_DBG
	default:
		pkiLogLevel = kritis3m_pki.KRITIS3M_PKI_LOG_LEVEL_WRN
	}

	err = kritis3m_pki.InitPKI(&kritis3m_pki.KRITIS3MPKIConfiguration{
		LogLevel:       int32(pkiLogLevel),
		LoggingEnabled: true,
	})
	if err != nil {
		asl.ASLFreeEndpoint(endpoint)
		asl.ASLshutdown()
		cancel()
		return nil, fmt.Errorf("failed to initialize PKI: %v", err)
	}

	// Create logger
	var logger est.Logger
	var logFile *os.File

	estLogLevel := cfg.Log.Level
	if cfg.Log.Format == "" {
		logger = alogger.New(os.Stderr, estLogLevel)
	} else {
		logFilePath := "/tmp/estserver.log" // Default log file path
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			asl.ASLFreeEndpoint(endpoint)
			asl.ASLshutdown()
			cancel()
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

	// Create CA
	ca, err := realca.Load(
		cfg.CA.Certificates,
		cfg.CA.PrivateKey,
		logger,
		pkcs11Config,
		validity,
	)
	if err != nil {
		if logFile != nil {
			logFile.Close()
		}
		asl.ASLFreeEndpoint(endpoint)
		cancel()
		return nil, fmt.Errorf("failed to create CA: %v", err)
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
		asl.ASLFreeEndpoint(endpoint)
		cancel()
		return nil, fmt.Errorf("failed to create new EST router: %v", err)
	}

	// Create ASL HTTP server
	aslServer := &aslhttpserver.ASLServer{
		Server: &http.Server{
			Addr:    cfg.TLS.ListenAddress,
			Handler: r,
			ConnContext: func(ctx context.Context, c net.Conn) context.Context {
				if aslConn, ok := c.(*aslListener.ASLConn); ok {
					if aslConn.TLSState != nil {
						// Attach the TLS state to the context
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
		server:     aslServer,
		endpoint:   endpoint,
		logFile:    logFile,
		cancelFunc: cancel,
		ctx:        ctx,
	}, nil
}

// Serve starts the EST server
func (e *ESTServer) Serve() error {
	go func() {
		err := e.server.ListenAndServeASLTLS()
		if err != nil && err != http.ErrServerClosed {
			zerolog_log.Error().Err(err).Msg("EST server error")
		}
	}()

	// Start a goroutine to monitor the context for cancellation
	go func() {
		<-e.ctx.Done()
		zerolog_log.Info().Msg("EST server context cancelled, shutting down")
		e.shutdownInternal()
	}()

	zerolog_log.Info().Msg("EST server started")
	return nil
}

// shutdownInternal handles the internal shutdown logic
func (e *ESTServer) shutdownInternal() {
	if e.server != nil && e.server.Server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := e.server.Server.Shutdown(ctx); err != nil {
			zerolog_log.Error().Err(err).Msg("Error shutting down EST server")
		}
	}

	if e.endpoint != nil {
		asl.ASLFreeEndpoint(e.endpoint)
	}

	if e.logFile != nil {
		e.logFile.Close()
	}
}

// Shutdown gracefully stops the EST server
func (e *ESTServer) Shutdown() {
	e.cancelFunc() // This will trigger the monitoring goroutine in Serve()

	// Wait a bit to allow the server to shut down gracefully
	time.Sleep(100 * time.Millisecond)

	// Call ASL shutdown after cancelling, in case it wasn't already called

	zerolog_log.Info().Msg("EST server stopped")
}
