package types

import (
	"errors"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
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

var ErrCannotParsePrefix = errors.New("cannot parse prefix")

// ASLKeyExchangeMethodToProto converts a string ASL key exchange method to the proto enum
func ASLKeyExchangeMethodToProto(method string) grpc_southbound.AslKeyexchangeMethod {
	if val, ok := grpc_southbound.AslKeyexchangeMethod_value[method]; ok {
		return grpc_southbound.AslKeyexchangeMethod(val)
	}
	return grpc_southbound.AslKeyexchangeMethod_ASL_KEX_DEFAULT
}
