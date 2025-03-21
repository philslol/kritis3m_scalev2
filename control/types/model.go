package types

import (
	"net"
	"time"

	"github.com/gofrs/uuid/v5"
)

// VersionState represents the possible states of a versioned entity
type VersionState string

const (
	VERSION_STATE_DRAFT              VersionState = "draft"
	VERSION_STATE_PENDING_DEPLOYMENT VersionState = "pending_deployment"
	VERSION_STATE_ACTIVE             VersionState = "active"
	VERSION_STATE_DISABLED           VersionState = "disabled"
)

type VersionTransitionStatus string

const (
	VersionTransitionPending  VersionTransitionStatus = "pending"
	VersionTransitionActive   VersionTransitionStatus = "active"
	VersionTransitionFailed   VersionTransitionStatus = "failed"
	VersionTransitionRollback VersionTransitionStatus = "rollback"
)

// TransactionType represents the PostgreSQL ENUM transaction_type.
type TransactionType string

const (
	TransactionTypeNodeUpdate    TransactionType = "node_update"
	TransactionTypeGroupUpdate   TransactionType = "group_update"
	TransactionTypeVersionUpdate TransactionType = "version_update"
)

type TransactionState string

const (
	TransactionStateError      TransactionState = "error"
	TransactionStateUnknown    TransactionState = "unknown"
	TransactionStatePublished  TransactionState = "published"
	TransactionStateReceived   TransactionState = "received"
	TransactionStateApplicable TransactionState = "applicable"
	TransactionStateApplied    TransactionState = "applied"
)

// Enum mapping for PostgreSQL (converts lowercase DB values to Go uppercase)
var VersionStateMap = map[string]int32{
	"draft":              0,
	"pending_deployment": 1,
	"active":             2,
	"disabled":           3,
}

type ProxyType string

const (
	PROXY_TYPE_NOT_SPECIFIED ProxyType = "not_specified"
	PROXY_TYPE_FORWARD       ProxyType = "forward"
	PROXY_TYPE_REVERSE       ProxyType = "reverse"
	PROXY_TYPE_TLSTLS        ProxyType = "tlstls"
)

var ProxyTypeMap = map[ProxyType]int32{
	"not_specified": 0,
	"forward":       1,
	"reverse":       2,
	"tlstls":        3,
}

type Node struct {
	ID           int        `json:"id"`
	SerialNumber string     `json:"serial_number"`
	NetworkIndex int        `json:"network_index"`
	Locality     string     `json:"locality"`
	LastSeen     *time.Time `json:"last_seen,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	CreatedBy    string     `json:"created_by"`
	VersionSetID uuid.UUID  `json:"version_set_id,omitempty"`
}

type Group struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	LogLevel           int       `json:"log_level"`
	EndpointConfigName string    `json:"endpoint_config_name,omitempty"`
	LegacyConfigName   string    `json:"legacy_config_name,omitempty"`
	VersionSetID       uuid.UUID `json:"version_set_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          string    `json:"created_by"`
}

type HardwareConfig struct {
	ID           int       `json:"id"`
	NodeSerial   string    `json:"node_serial"`
	Device       string    `json:"device"`
	IPCIDR       net.IPNet `json:"ip_cidr"`
	VersionSetID uuid.UUID `json:"version_set_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CreatedBy    string    `json:"created_by"`
}

type Proxy struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	NodeSerial         string    `json:"node_serial"`
	GroupName          string    `json:"group_name"`
	State              bool      `json:"state"`
	ProxyType          ProxyType `json:"proxy_type"`
	ServerEndpointAddr string    `json:"server_endpoint_addr"`
	ClientEndpointAddr string    `json:"client_endpoint_addr"`
	VersionSetID       uuid.UUID `json:"version_set_id,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          string    `json:"created_by"`
}

type VersionSet struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description *string      `json:"description"`
	State       VersionState `json:"state"`
	CreatedAt   time.Time    `json:"created_at"`
	ActivatedAt *time.Time   `json:"activated_at"`
	DisabledAt  *time.Time   `json:"disabled_at"`
	CreatedBy   string       `json:"created_by"`
	Metadata    []byte       `json:"metadata"` // JSONB is stored as a byte slice
}

type EndpointConfig struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	MutualAuth           bool      `json:"mutual_auth"`
	NoEncryption         bool      `json:"no_encryption"`
	ASLKeyExchangeMethod string    `json:"asl_key_exchange_method"`
	Cipher               *string   `json:"cipher"`
	VersionSetID         uuid.UUID `json:"version_set_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedBy            string    `json:"created_by"`
}

// Transaction represents the transactions table.
type Transaction struct {
	ID                  int             `json:"id"`
	Type                TransactionType `json:"type"`
	VersionTransitionID *int            `json:"version_transition_id,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	CompletedAt         *time.Time      `json:"completed_at,omitempty"`
	Description         *string         `json:"description,omitempty"`
}

// TransactionLog represents the transaction_log table.
type NodeTransactionLog struct {
	ID            int              `json:"id"`
	TransactionID int              `json:"transaction_id"`
	NodeSerial    string           `json:"node_serial"`
	VersionSetID  uuid.UUID        `json:"version_set_id"`
	State         TransactionState `json:"state"`
	Timestamp     time.Time        `json:"timestamp"`
	Metadata      []byte           `json:"metadata"`
}

// VersionTransition represents the version_transitions table.
type VersionTransition struct {
	ID                    int                     `json:"id"`
	FromVersionTransition *int                    `json:"from_version_transition"`
	ToVersionSetID        uuid.UUID               `json:"to_version_id"`
	TransactionID         int                     `json:"transaction_id"`
	Status                VersionTransitionStatus `json:"status"`
	StartedAt             time.Time               `json:"started_at"`
	CompletedAt           *time.Time              `json:"completed_at"`
	CreatedBy             string                  `json:"created_by"`
	Metadata              []byte                  `json:"metadata"`
}
