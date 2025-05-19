package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/kritis3m_pki"
	"github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_est/lib/realca"
	"github.com/philslol/kritis3m_scalev2/control/util"

	asl "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	defaultOIDCExpiryTime               = 180 * 24 * time.Hour // 180 Days
	maxDuration           time.Duration = 1<<63 - 1
)

// Config contains the initial Headscale configuration.
type Config struct {
	Log     LogConfig
	Logfile string

	Database DatabaseConfig

	Log_Database DatabaseConfig
	// CLI          CLIConfig
	ACL       ACLConfig
	ASLConfig asl.ASLConfig

	Broker       BrokerConfig
	ControlPlane ControlPlaneConfig
	ESTServer    ESTServerConfig

	CLILog   LogConfig
	NodeLog  LogConfig
	HelloLog LogConfig

	CliConfig CliConfig
}

type ACLConfig struct {
	PolicyPath string
}

type BrokerConfig struct {
	Adress         string
	Log            LogConfig
	EndpointConfig asl.EndpointConfig
	TcpOnly        bool
}

type ControlPlaneConfig struct {
	Address        string
	Log            LogConfig
	EndpointConfig asl.EndpointConfig
	TcpOnly        bool
}

type LogConfig struct {
	Format string
	Level  zerolog.Level
	File   string
}

// Config holds the database configuration
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DatabaseName string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	LogConfig    LogConfig
}

type CliConfig struct {
	Timeout    time.Duration
	ServerAddr string
}

// ESTServerConfig holds the configuration for the EST server
type ESTServerConfig struct {
	ServerAddress  string
	CA             CAConfig
	EndpointConfig asl.EndpointConfig
	AllowedHosts   []string
	HealthCheckPwd string
	RateLimit      int
	Timeout        int
	Log            LogConfig
	ASLConfig      asl.ASLConfig
}

// ESTEndpointConfig holds the endpoint configuration
type ESTEndpointConfig struct {
	ListenAddress        string
	Certificates         string
	PrivateKey           string
	RootCerts            []string
	MutualAuthentication bool
	Ciphersuites         []string
	KeylogFile           string
	PKCS11               PKCS11Module
}

// CAConfig holds the CA configuration
type CAConfig struct {
	Backends       []PKIBackendConfig
	DefaultBackend *PKIBackendConfig
	Validity       int
}

// PKIBackendConfig holds configuration for a PKI backend
type PKIBackendConfig = realca.PKIBackendConfig

// TLSConfig holds the TLS configuration
type TLSConfig struct {
	ListenAddress string
	Certificates  string
	PrivateKey    string
	ClientCAs     []string
	ASLEndpoint   ASLEndpointConfig
	PKCS11Module  PKCS11Module
}

// ASLEndpointConfig holds ASL-specific TLS configuration
type ASLEndpointConfig struct {
	MutualAuthentication bool
	Ciphersuites         []string
	KeylogFile           string
}

// PKCS11Module holds PKCS11 module configuration
type PKCS11Module struct {
	Path string
	Pin  string
}

func toASLKeyExchangeMethod(s string) (asl.ASLKeyExchangeMethod, error) {

	var kex asl.ASLKeyExchangeMethod
	if s == "KEX_DEFAULT" {
		kex = asl.KEX_DEFAULT
	} else if s == "KEX_CLASSIC_ECDHE_256" {
		kex = asl.KEX_CLASSIC_ECDHE_256
	} else if s == "KEX_CLASSIC_ECDHE_384" {
		kex = asl.KEX_CLASSIC_ECDHE_384
	} else if s == "KEX_CLASSIC_ECDHE_521" {
		kex = asl.KEX_CLASSIC_ECDHE_521
	} else if s == "KEX_CLASSIC_X25519" {
		kex = asl.KEX_CLASSIC_X25519
	} else if s == "KEX_CLASSIC_X448" {
		kex = asl.KEX_CLASSIC_X448
	} else if s == "KEX_PQC_MLKEM_512" {
		kex = asl.KEX_PQC_MLKEM_512
	} else if s == "KEX_PQC_MLKEM_768" {
		kex = asl.KEX_PQC_MLKEM_768
	} else if s == "KEX_PQC_MLKEM_1024" {
		kex = asl.KEX_PQC_MLKEM_1024
	} else if s == "KEX_HYBRID_ECDHE_256_MLKEM_512" {
		kex = asl.KEX_HYBRID_ECDHE_256_MLKEM_512
	} else if s == "KEX_HYBRID_ECDHE_384_MLKEM_768" {
		kex = asl.KEX_HYBRID_ECDHE_384_MLKEM_768
	} else if s == "KEX_HYBRID_ECDHE_521_MLKEM_1024" {
		kex = asl.KEX_HYBRID_ECDHE_521_MLKEM_1024
	} else if s == "KEX_HYBRID_X25519_MLKEM_512" {
		kex = asl.KEX_HYBRID_X25519_MLKEM_512
	} else if s == "KEX_HYBRID_X25519_MLKEM_768" {
		kex = asl.KEX_HYBRID_X25519_MLKEM_768
	} else if s == "KEX_HYBRID_X448_MLKEM_768" {
		kex = asl.KEX_HYBRID_X448_MLKEM_768
	} else {
		return -1, fmt.Errorf("unknown key exchange method provided")
	}

	return kex, nil
}
func parse_Log(basepath string) LogConfig {
	var log_config LogConfig

	log_config.Format = viper.GetString(fmt.Sprintf("%s.%s", basepath, "format"))
	log_config.Level = zerolog.Level(viper.GetInt(fmt.Sprintf("%s.%s", basepath, "log_level")))
	log_config.File = viper.GetString(fmt.Sprintf("%s.%s", basepath, "file"))
	return log_config
}

func parse_ASLEndpointConfig(basepath string) (*asl.EndpointConfig, error) {
	var kex asl.ASLKeyExchangeMethod
	var pkcs11 asl.PKCS11ASL
	var root asl.RootCertificates
	var device asl.DeviceCertificateChain
	var private asl.PrivateKey

	pkcs11.Path = viper.GetString(fmt.Sprintf("%s.%s", basepath, "pkcs11.path"))
	pkcs11.Pin = viper.GetString(fmt.Sprintf("%s.%s", basepath, "pkcs11.pin"))
	kex_string := viper.GetString(fmt.Sprintf("%s.%s", basepath, "key_exchange_method"))
	kex, err := toASLKeyExchangeMethod(kex_string)
	if err != nil {
		log.Err(err)
		return nil, err
	}

	// Get keys
	private.Path = viper.GetString(fmt.Sprintf("%s.%s", basepath, "private_key"))
	private.AdditionalKeyPath = viper.GetString(fmt.Sprintf("%s.%s", basepath, "alt_private_key"))
	device.Path = viper.GetString(fmt.Sprintf("%s.%s", basepath, "device_cert"))
	root.Paths = viper.GetStringSlice(fmt.Sprintf("%s.%s", basepath, "root_certs"))

	if len(root.Paths) == 0 || device.Path == "" || private.Path == "" {
		err := fmt.Errorf("either device or root or private key is empty")
		log.Err(err)
		return nil, err
	}

	var ep_config asl.EndpointConfig = asl.EndpointConfig{
		KeylogFile:             viper.GetString(fmt.Sprintf("%s.%s", basepath, "key_log_file")),
		MutualAuthentication:   viper.GetBool(fmt.Sprintf("%s.%s", basepath, "mutual_authentication")),
		ASLKeyExchangeMethod:   kex,
		PKCS11:                 pkcs11,
		RootCertificates:       root,
		DeviceCertificateChain: device,
		PrivateKey:             private,
		Ciphersuites:           viper.GetStringSlice(fmt.Sprintf("%s.%s", basepath, "ciphersuites")),
	}
	return &ep_config, nil
}

func GetDatabaseConfig() (*DatabaseConfig, error) {
	// Unmarshal the specific section into the struct
	var database_config DatabaseConfig

	if err := viper.Sub("database.postgres").Unmarshal(&database_config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	database_config.LogConfig = parse_Log("database.log")
	return &database_config, nil
}

func LoadConfig(path string, isFile bool) error {
	if isFile {
		viper.SetConfigFile(path)
	} else {

		log.Warn().Msg("Failed to read configuration from disk, no config file provided or path is incorrect")
		return nil

	}

	viper.SetEnvPrefix("kritis3m_scale")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("log.level", "info")
	viper.SetDefault("log.format", TextLogFormat)

	if err := viper.ReadInConfig(); err != nil {
		log.Warn().Err(err).Msg("Failed to read configuration from disk")

		return fmt.Errorf("fatal error reading config file: %w", err)
	}

	// Collect any validation errors and return them all at once
	var errorText string

	if errorText != "" {
		// nolint
		return errors.New(strings.TrimSuffix(errorText, "\n"))
	} else {
		return nil
	}
}

func GetASLConfig() asl.ASLConfig {
	return asl.ASLConfig{
		LoggingEnabled: viper.GetBool("asl_config.logging_enabled"),
		LogLevel:       viper.GetInt32("asl_config.log_level"),
	}
}

func GetACLConfig() ACLConfig {
	policyPath := viper.GetString("acl_policy_path")
	policyPath = util.AbsolutePathFromConfigPath(policyPath)

	return ACLConfig{
		PolicyPath: policyPath,
	}
}

func GetBrokerConfig() (*BrokerConfig, error) {
	var broker_config BrokerConfig

	log := parse_Log("broker_config.log")
	ep, err := parse_ASLEndpointConfig("broker_config.endpoint_config")
	if err != nil {
		return nil, err
	}

	broker_config.TcpOnly = viper.GetBool("broker_config.tcp_only")

	broker_config.Log = log
	broker_config.EndpointConfig = *ep
	broker_config.Adress = viper.GetString("broker_config.address")
	if broker_config.Adress == "" {
		return nil, fmt.Errorf("no address specified for broker adress")
	}

	return &broker_config, nil
}

func GetControlPlaneConfig() (*ControlPlaneConfig, error) {
	var control_plane_config ControlPlaneConfig

	log := parse_Log("control_plane_config.log")
	ep, err := parse_ASLEndpointConfig("control_plane_config.endpoint_config")
	if err != nil {
		return nil, err
	}
	control_plane_config.TcpOnly = viper.GetBool("control_plane_config.tcp_only")

	control_plane_config.Log = log
	control_plane_config.EndpointConfig = *ep
	control_plane_config.Address = viper.GetString("control_plane_config.server_address")
	if control_plane_config.Address == "" {
		return nil, fmt.Errorf("no address specified for control plane address")
	}

	return &control_plane_config, nil
}

func GetESTServerConfig() (*ESTServerConfig, error) {
	var estConfig ESTServerConfig

	// Parse CA config
	caConfig := viper.Sub("est_server_config.ca")
	if caConfig == nil {
		return nil, fmt.Errorf("missing CA configuration")
	}

	// Parse general EST server config
	estConfig.AllowedHosts = viper.GetStringSlice("est_server_config.allowed_hosts")
	estConfig.HealthCheckPwd = viper.GetString("est_server_config.healthcheck_password")
	estConfig.RateLimit = viper.GetInt("est_server_config.rate_limit")
	estConfig.Timeout = viper.GetInt("est_server_config.timeout")
	estConfig.Log = parse_Log("est_server_config.log")
	estConfig.ServerAddress = viper.GetString("est_server_config.server_address")

	ep, err := parse_ASLEndpointConfig("est_server_config.endpoint_config")
	if err != nil {
		return nil, err
	}
	estConfig.EndpointConfig = *ep

	// Parse backends

	// First unmarshal the config
	var backends []PKIBackendConfig
	if err := caConfig.UnmarshalKey("backends", &backends); err != nil {
		return nil, fmt.Errorf("failed to parse CA backends: %v", err)
	}

	// Then update each backend in place using index notation
	for i := range backends {
		backends[i].APS = viper.GetString(fmt.Sprintf("est_server_config.ca.backends.%d.aps", i))
		backends[i].Module = &kritis3m_pki.PKCS11Module{
			Path: viper.GetString(fmt.Sprintf("est_server_config.ca.backends.%d.pkcs11_module.path", i)),
			Pin:  viper.GetString(fmt.Sprintf("est_server_config.ca.backends.%d.pkcs11_module.pin", i)),
			Slot: viper.GetInt(fmt.Sprintf("est_server_config.ca.backends.%d.pkcs11_module.slot", i)),
		}
		backends[i].Certificates = viper.GetString(fmt.Sprintf("est_server_config.ca.backends.%d.certificates", i))
		backends[i].PrivateKey = viper.GetString(fmt.Sprintf("est_server_config.ca.backends.%d.private_key", i))
	}

	// Parse default backend
	var defaultBackend PKIBackendConfig
	defaultBackend.Module = &kritis3m_pki.PKCS11Module{
		Path: viper.GetString("est_server_config.ca.default_backend.pkcs11_module.path"),
		Pin:  viper.GetString("est_server_config.ca.default_backend.pkcs11_module.pin"),
		Slot: viper.GetInt("est_server_config.ca.default_backend.pkcs11_module.slot"),
	}
	defaultBackend.Certificates = viper.GetString("est_server_config.ca.default_backend.certificates")
	defaultBackend.PrivateKey = viper.GetString("est_server_config.ca.default_backend.private_key")
	defaultBackend.APS = viper.GetString("est_server_config.ca.default_backend.aps")

	estConfig.CA = CAConfig{
		Backends:       backends,
		DefaultBackend: &defaultBackend,
		Validity:       caConfig.GetInt("validity"),
	}

	estConfig.ASLConfig = asl.ASLConfig{
		LoggingEnabled: viper.GetBool("est_server_config.asl_config.logging_enabled"),
		LogLevel:       viper.GetInt32("est_server_config.asl_config.log_level"),
	}

	return &estConfig, nil
}

func GetKritis3mScaleConfig() (*Config, error) {
	ctrl_plane_cfg, err := GetControlPlaneConfig()
	if err != nil {
		return nil, err
	}
	broker, err := GetBrokerConfig()
	if err != nil {
		return nil, err
	}
	database_config, err := GetDatabaseConfig()
	if err != nil {
		return nil, err
	}

	estServer, err := GetESTServerConfig()
	if err != nil {
		return nil, err
	}
	cli_log := parse_Log("cli_log")

	return &Config{
		Logfile:      viper.GetString("log_file"),
		ACL:          GetACLConfig(),
		ASLConfig:    GetASLConfig(),
		Database:     *database_config,
		Broker:       *broker,
		ControlPlane: *ctrl_plane_cfg,
		ESTServer:    *estServer,
		Log:          parse_Log(""),
		CliConfig:    GetCliConfig(),
		CLILog:       cli_log,
		NodeLog:      parse_Log("node_log"),
		HelloLog:     parse_Log("hello_log"),
	}, nil
}

func GetCliConfig() CliConfig {
	timeout := viper.GetDuration("cli_timeout_s")
	//convert to seconds
	timeout = timeout * time.Second
	serverAddr := viper.GetString("grpc_listen_addr")
	return CliConfig{
		Timeout:    timeout,
		ServerAddr: serverAddr,
	}

}
