package types

import (
	"net"
	"time"

	"github.com/gofrs/uuid"
)

// VersionState represents the possible states of a versioned entity
type VersionState string

const (
	VersionStateDraft     VersionState = "draft"
	VersionStatePublished VersionState = "published"
	VersionStateArchived  VersionState = "archived"
)

type Node struct {
	ID           int          `json:"id"`
	SerialNumber string       `json:"serial_number"`
	NetworkIndex int          `json:"network_index"`
	Locality     string       `json:"locality"`
	LastSeen     *time.Time   `json:"last_seen,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	CreatedBy    string       `json:"created_by"`
	VersionSetID *uuid.UUID   `json:"version_set_id,omitempty"`
	State        VersionState `json:"state"`
}

type Group struct {
	ID               int          `json:"id"`
	Name             string       `json:"name"`
	LogLevel         int          `json:"log_level"`
	EndpointConfigID *int         `json:"endpoint_config_id,omitempty"`
	LegacyConfigID   *int         `json:"legacy_config_id,omitempty"`
	State            VersionState `json:"state"`
	VersionSetID     *uuid.UUID   `json:"version_set_id,omitempty"`
	CreatedAt        time.Time    `json:"created_at"`
	UpdatedAt        time.Time    `json:"updated_at"`
	CreatedBy        string       `json:"created_by"`
}

type HardwareConfig struct {
	ID           int          `json:"id"`
	NodeID       *int         `json:"node_id,omitempty"`
	Device       string       `json:"device"`
	IPCIDR       net.IPNet    `json:"ip_cidr"`
	VersionSetID *uuid.UUID   `json:"version_set_id,omitempty"`
	State        VersionState `json:"state"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	CreatedBy    string       `json:"created_by"`
}

type Proxy struct {
	ID                 int        `json:"id"`
	NodeID             *int       `json:"node_id,omitempty"`
	GroupID            *int       `json:"group_id,omitempty"`
	State              bool       `json:"state"`
	ProxyType          string     `json:"proxy_type"`
	ServerEndpointAddr string     `json:"server_endpoint_addr"`
	ClientEndpointAddr string     `json:"client_endpoint_addr"`
	VersionSetID       *uuid.UUID `json:"version_set_id,omitempty"`
	VersionState       string     `json:"version_state"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CreatedBy          string     `json:"created_by"`
}

type VersionSet struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description"`
	State       string     `json:"state"`
	CreatedAt   time.Time  `json:"created_at"`
	ActivatedAt *time.Time `json:"activated_at"`
	DisabledAt  *time.Time `json:"disabled_at"`
	CreatedBy   string     `json:"created_by"`
	Metadata    []byte     `json:"metadata"` // JSONB is stored as a byte slice
}

type VersionTransition struct {
	ID            string     `json:"id"`
	FromVersionID *string    `json:"from_version_id"`
	ToVersionID   string     `json:"to_version_id"`
	Status        string     `json:"status"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at"`
	CreatedBy     string     `json:"created_by"`
	Metadata      []byte     `json:"metadata"`
}

type EndpointConfig struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	MutualAuth           bool      `json:"mutual_auth"`
	NoEncryption         bool      `json:"no_encryption"`
	ASLKeyExchangeMethod string    `json:"asl_key_exchange_method"`
	Cipher               *string   `json:"cipher"`
	State                string    `json:"state"`
	VersionSetID         *string   `json:"version_set_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedBy            string    `json:"created_by"`
}
