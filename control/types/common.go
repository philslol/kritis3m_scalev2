package types

import (
	"errors"
	"os"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/rs/zerolog"
)

const (
	SelfUpdateIdentifier = "self-update"
	DatabasePostgres     = "postgres"
	DatabaseSqlite       = "sqlite3"
)

// api
const (
	DataNotound = iota + 1
	UnkonwnError
	InvalidRequest
	Unauthorized
)

type Operationtype int32
type Statustype int32
type PollingInterval_min int32

const (
	//status
	Config_up_to_date    Statustype = 0
	New_config_available Statustype = 1
)

// see you again
const (
	Short_interval  PollingInterval_min = 1
	Medium_interval PollingInterval_min = 60      //every hour
	Long_interval   PollingInterval_min = 60 * 24 //every day
)

// operation request
const (
	Nothing         Operationtype = 0
	Request_new_cfg Operationtype = 1
	Shut_down       Operationtype = 2
	Restart         Operationtype = 3
)

func CreateLogger(module string, log_level zerolog.Level, log_file string) zerolog.Logger {
	if log_file == "" {
		return zerolog.New(os.Stdout).Level(log_level).With().Str("module", module).Timestamp().Logger()
	}

	// Create a file writer that's safe for concurrent access
	file, err := os.OpenFile(log_file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// If we can't open the log file, fall back to stdout
		return zerolog.New(os.Stdout).Level(log_level).With().
			Str("module", module).
			Str("error", "Failed to open log file: "+err.Error()).
			Timestamp().Logger()
	}

	// Create a multi-writer to write to both file and stdout
	multi := zerolog.MultiLevelWriter(os.Stdout, file)

	return zerolog.New(multi).Level(log_level).With().Str("module", module).Timestamp().Logger()
}

var ErrCannotParsePrefix = errors.New("cannot parse prefix")

// ASLKeyExchangeMethodToProto converts a string ASL key exchange method to the proto enum
func ASLKeyExchangeMethodToProto(method string) grpc_southbound.AslKeyexchangeMethod {
	if val, ok := grpc_southbound.AslKeyexchangeMethod_value[method]; ok {
		return grpc_southbound.AslKeyexchangeMethod(val)
	}
	return grpc_southbound.AslKeyexchangeMethod_ASL_KEX_DEFAULT
}
