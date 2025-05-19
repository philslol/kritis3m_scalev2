package southbound

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	grpc_est "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/est"
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/db"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
)

// SouthboundService handles communication with the control plane
type SouthboundService struct {
	db   *db.StateManager
	addr string
	grpc_southbound.UnimplementedSouthboundServer
	grpc_est.UnimplementedEstServiceServer
}

// NewSouthbound creates a new instance of SouthboundService
func NewSouthbound(db *db.StateManager, addr string) *SouthboundService {
	return &SouthboundService{
		db:   db,
		addr: addr,
	}
}

type LogService struct {
	filepath string
	db       *db.StateManager
	addr     string
	logger   zerolog.Logger
}

type HelloService struct {
	db     *db.StateManager
	addr   string
	logger zerolog.Logger
}

func NewHelloService(db *db.StateManager, addr string, log_config types.LogConfig) *HelloService {
	return &HelloService{
		db:     db,
		addr:   addr,
		logger: types.CreateLogger("hello", log_config.Level, log_config.File),
	}
}

func NewLogService(db *db.StateManager, addr string, log_config types.LogConfig) *LogService {
	return &LogService{
		db:       db,
		addr:     addr,
		filepath: log_config.File,
		logger:   types.CreateLogger("log", log_config.Level, log_config.File),
	}
}

func (ls *LogService) LogNodeTransaction(ctx context.Context) error {
	// Open filepath and create new zerologger instance for that path
	file, err := os.OpenFile(ls.filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	client, conn, err := getControlPlaneClient(ls.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	stream, err := client.Log(ctx, &empty.Empty{})
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)

	go func() {
		for {
			select {
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			default:
				log_response, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						ls.logger.Info().Msg("Log stream closed")
						errChan <- nil
						return
					}
					ls.logger.Error().Err(err).Msg("Error receiving log response")
					errChan <- err
					return
				}

				log_response.Message = strings.ReplaceAll(log_response.Message, "\n", " ")

				// Log with appropriate level
				msg := fmt.Sprintf("node: %s,module: %s: msg: %s", log_response.SerialNumber, *log_response.Module, log_response.Message)

				if log_response.Level != nil {
					switch *log_response.Level {
					case 0:
						ls.logger.Trace().Msg(msg)
					case 1:
						ls.logger.Error().Msg(msg)
					case 2:
						ls.logger.Warn().Msg(msg)
					case 3:
						ls.logger.Info().Msg(msg)
					case 4:
						ls.logger.Debug().Msg(msg)
					default:
						ls.logger.Info().Msg(msg)
					}
				} else {
					// Default to Info level when level not provided
					ls.logger.Info().Msg(msg)
				}
			}
		}
	}()

	// Wait for context cancellation or error from goroutine
	select {
	case <-ctx.Done():
		ls.logger.Info().Msg("Log service context cancelled")
		return ctx.Err()
	case err := <-errChan:
		ls.logger.Error().Err(err).Msg("Log service error")
		return err
	}
}

func (hs *HelloService) Hello(ctx context.Context) error {
	errChan := make(chan error, 1)
	client, conn, err := getControlPlaneClient(hs.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	stream, err := client.Hello(ctx, &empty.Empty{})
	if err != nil {
		hs.logger.Error().Err(err).Msg("Error creating hello stream")
		return err
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				response, err := stream.Recv()
				if err != nil {
					if err == io.EOF {
						hs.logger.Info().Msg("Hello stream closed")
						errChan <- nil
						return
					}
					hs.logger.Error().Err(err).Msg("Error receiving hello response")
					errChan <- err
					return
				}
				// Use a parameterized query instead of string interpolation
				where_string := fmt.Sprintf("serial_number = '%s'", response.SerialNumber)
				db_context, db_cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer db_cancel()

				//it is not intendet to close hello service, when db has an error
				err = hs.db.UpdateWhere(db_context, "nodes", map[string]any{"last_seen": time.Now()}, where_string)
				if err != nil {
					hs.logger.Error().Err(err).Msg("Error updating node last seen")
				}
			}
		}
	}()
	select {
	case <-ctx.Done():
		hs.logger.Info().Msg("Hello service context cancelled")
		return ctx.Err()
	case err := <-errChan:
		hs.logger.Error().Err(err).Msg("Hello service error")
		return err
	}
}
