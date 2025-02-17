package types

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type EndpointConfig struct {
	ID                   int       `json:"id"`
	TransactionID        string    `json:"transaction_id"`
	Name                 string    `json:"name"`
	MutualAuth           bool      `json:"mutual_auth"`
	NoEncryption         bool      `json:"no_encryption"`
	ASLKeyExchangeMethod string    `json:"asl_key_exchange_method"`
	Cipher               string    `json:"cipher"`
	Status               string    `json:"status"`
	Version              int       `json:"version"`
	PreviousVersionID    *int      `json:"previous_version_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	CreatedBy            string    `json:"created_by"`
}

type Proxy struct {
	ID                 int       `json:"id"`
	TransactionID      string    `json:"transaction_id"`
	NodeID             int       `json:"node_id"`
	GroupID            int       `json:"group_id"`
	State              bool      `json:"state"`
	ProxyType          string    `json:"proxy_type"`
	ServerEndpointAddr string    `json:"server_endpoint_addr"`
	ClientEndpointAddr string    `json:"client_endpoint_addr"`
	Status             string    `json:"status"`
	Version            int       `json:"version"`
	PreviousVersionID  *int      `json:"previous_version_id"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	CreatedBy          string    `json:"created_by"`
}

type HardwareConfig struct {
	ID                int         `json:"id"`
	NodeID            int         `json:"node_id"`
	TransactionID     string      `json:"transaction_id"`
	Device            string      `json:"device"`
	IPCIDR            pgtype.Inet `json:"ip_cidr"`
	Status            string      `json:"status"`
	Version           int         `json:"version"`
	PreviousVersionID *int        `json:"previous_version_id"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	CreatedBy         string      `json:"created_by"`
}

type Group struct {
	ID                int       `json:"id"`
	TransactionID     string    `json:"transaction_id"`
	Name              string    `json:"name"`
	LogLevel          int       `json:"log_level"`
	EndpointConfigID  *int      `json:"endpoint_config_id"`
	LegacyConfigID    *int      `json:"legacy_config_id"`
	Status            string    `json:"status"`
	Version           int       `json:"version"`
	PreviousVersionID *int      `json:"previous_version_id"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CreatedBy         string    `json:"created_by"`
}

type Node struct {
	ID           int                `json:"id"`
	SerialNumber string             `json:"serial_number"`
	NetworkIndex int                `json:"network_index"`
	Locality     string             `json:"locality"`
	LastSeen     pgtype.Timestamptz `json:"last_seen"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
	CreatedBy    string             `json:"created_by"`
}
