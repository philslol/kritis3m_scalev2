package southbound

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/philslol/kritis3m_scalev2/control/db"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SouthboundService handles communication with the control plane
type SouthboundService struct {
	db   *db.StateManager
	addr string
	v1.UnimplementedSouthboundServer
}

// NewSouthbound creates a new instance of SouthboundService
func NewSouthbound(db *db.StateManager, addr string) *SouthboundService {
	//create new controlplane client
	return &SouthboundService{
		db:   db,
		addr: addr,
	}
}

type LogService struct {
	filepath string
	db       *db.StateManager
	addr     string
}

type HelloService struct {
	db   *db.StateManager
	addr string
}

func NewHelloService(db *db.StateManager, addr string) *HelloService {
	return &HelloService{
		db:   db,
		addr: addr,
	}
}

func NewLogService(db *db.StateManager, addr string, logfile string) *LogService {
	return &LogService{
		db:       db,
		addr:     addr,
		filepath: logfile,
	}
}

func (ls *LogService) LogNodeTransaction(ctx context.Context) error {
	// Open filepath and create new zerologger instance for that path
	file, err := os.OpenFile(ls.filepath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	logger := zerolog.New(file).With().Timestamp().Logger()

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
						logger.Info().Msg("Log stream closed")
						errChan <- nil
						return
					}
					logger.Error().Err(err).Msg("Error receiving log response")
					errChan <- err
					return
				}

				// Create event with serial number
				event := logger.With().Str("serial_number", log_response.SerialNumber)

				// Add module if available
				if log_response.Module != nil {
					event = event.Str("module", *log_response.Module)
				}
				lg := event.Logger()

				// Log with appropriate level
				if log_response.Level != nil {
					switch *log_response.Level {
					case 0:
						lg.Trace().Msg(log_response.Message)
					case 1:
						lg.Debug().Msg(log_response.Message)
					case 2:
						lg.Info().Msg(log_response.Message)
					case 3:
						lg.Warn().Msg(log_response.Message)
					case 4:
						lg.Error().Msg(log_response.Message)
					case 5:
						lg.Fatal().Msg(log_response.Message)
					default:
						lg.Info().Msg(log_response.Message)
					}
				} else {
					// Default to Info level when level not provided
					lg.Info().Msg(log_response.Message)
				}
			}
		}
	}()

	// Wait for context cancellation or error from goroutine
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
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
		log.Error().Err(err).Msg("Error creating hello stream")
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
						log.Info().Msg("Hello stream closed")
						errChan <- nil
						return
					}
					log.Error().Err(err).Msg("Error receiving hello response")
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
					log.Error().Err(err).Msg("Error updating node last seen")
				}
			}
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
