package types

import (
	"errors"
	"fmt"
	"github.com/philslol/kritis3m_scalev2/control/util"
	"os"
	"strings"
	"time"

	asl "github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

const (
	defaultOIDCExpiryTime               = 180 * 24 * time.Hour // 180 Days
	maxDuration           time.Duration = 1<<63 - 1
)

// Endpoint config -> which must be converted to asl'S EndpointConfig
type KS_EndpointConfig struct {
	MutualAuthentication        bool
	NoEncryption                bool
	ASLKeyExchangeMethod        string
	SecureElementMiddlewarePath string
	Pin                         string
	HybridSignatureMode         string
	DeviceCertificateChain      string
	PrivateKey                  KS_PrivateKeyConfig
	RootCertificate             string
	KeylogFile                  string
}
type KS_PrivateKeyConfig struct {
	PrivateKey1Path string
	PrivateKey2Path string
}

// Config contains the initial Headscale configuration.
type Config struct {
	Log          LogConfig
	Database     DatabaseConfig
	Log_Database DatabaseConfig
	// CLI          CLIConfig
	ACL        ACLConfig
	NodeServer NodeServerConfig
	ASLConfig  asl.ASLConfig
}

type SqliteConfig struct {
	Path string
}

type DatabaseConfig struct {
	// Type sets the database type, either "sqlite3" or "postgres"
	Type  string
	Debug bool
	Level zerolog.Level

	Sqlite SqliteConfig
}

type NodeServerConfig struct {
	Address      string
	AddressHTTP  string
	ASL_Config   asl.ASLConfig
	ASL_Endpoint asl.EndpointConfig
	Log          LogConfig
	GinMode      string
}

// type CLIConfig struct {
// 	Address  string
// 	Timeout  time.Duration
// 	Insecure bool
// }

type ACLConfig struct {
	PolicyPath string
}

type LogConfig struct {
	Format string
	Level  zerolog.Level
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

func parse_endpoint(ep_yaml *KS_EndpointConfig) asl.EndpointConfig {
	var endpoint_config asl.EndpointConfig
	endpoint_config.MutualAuthentication = ep_yaml.MutualAuthentication
	endpoint_config.NoEncryption = ep_yaml.NoEncryption
	ex := ep_yaml.ASLKeyExchangeMethod
	if ex == "KEX_DEFAULT" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_DEFAULT
	} else if ex == "KEX_CLASSIC_ECDHE_256" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_CLASSIC_ECDHE_256
	} else if ex == "KEX_CLASSIC_ECDHE_384" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_CLASSIC_ECDHE_384
	} else if ex == "KEX_CLASSIC_ECDHE_521" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_CLASSIC_ECDHE_521
	} else if ex == "KEX_CLASSIC_X25519" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_CLASSIC_X25519
	} else if ex == "KEX_CLASSIC_X448" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_CLASSIC_X448
	} else if ex == "KEX_PQC_MLKEM_512" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_PQC_MLKEM_512
	} else if ex == "KEX_PQC_MLKEM_768" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_PQC_MLKEM_768
	} else if ex == "KEX_PQC_MLKEM_1024" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_PQC_MLKEM_1024
	} else if ex == "KEX_HYBRID_ECDHE_256_MLKEM_512" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_HYBRID_ECDHE_256_MLKEM_512
	} else if ex == "KEX_HYBRID_ECDHE_384_MLKEM_768" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_HYBRID_ECDHE_384_MLKEM_768
	} else if ex == "KEX_HYBRID_ECDHE_521_MLKEM_1024" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_HYBRID_ECDHE_521_MLKEM_1024
	} else if ex == "KEX_HYBRID_X25519_MLKEM_512" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_HYBRID_X25519_MLKEM_512
	} else if ex == "KEX_HYBRID_X25519_MLKEM_768" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_HYBRID_X25519_MLKEM_768
	} else if ex == "KEX_HYBRID_X448_MLKEM_768" {
		endpoint_config.ASLKeyExchangeMethod = asl.KEX_HYBRID_X448_MLKEM_768
	} else {
		panic("wrong format of key exchange method")
	}

	//check if path exists
	if ep_yaml.SecureElementMiddlewarePath != "" {
		if _, err := os.Stat(ep_yaml.SecureElementMiddlewarePath); errors.Is(err, os.ErrNotExist) {
			panic("SecureElementMiddlewarePath does not exist")
		}
	}
	//hybrid signature mode
	// mode := ep_yaml.HybridSignatureMode
	// if mode == "HYBRID_SIGNATURE_MODE_DEFAULT" {
	// 	endpoint_config.HybridSignatureMode = asl.HYBRID_SIGNATURE_MODE_DEFAULT
	// } else if mode == "HYBRID_SIGNATURE_MODE_NATIVE" {
	// 	endpoint_config.HybridSignatureMode = asl.HYBRID_SIGNATURE_MODE_NATIVE
	// } else if mode == "HYBRID_SIGNATURE_MODE_ALTERNATIVE" {
	// 	endpoint_config.HybridSignatureMode = asl.HYBRID_SIGNATURE_MODE_ALTERNATIVE
	// } else if mode == "HYBRID_SIGNATURE_MODE_BOTH" {
	// 	endpoint_config.HybridSignatureMode = asl.HYBRID_SIGNATURE_MODE_BOTH
	// } else if mode == "" {
	// 	panic("signature not provided")
	// } else {
	// 	panic("signature mode bad format ")
	// }
	//device cert:
	//check if path exists
	if ep_yaml.DeviceCertificateChain != "" {
		if _, err := os.Stat(ep_yaml.DeviceCertificateChain); errors.Is(err, os.ErrNotExist) {
			panic("device cert path does not exist")
		} else {
			endpoint_config.DeviceCertificateChain.Path = ep_yaml.DeviceCertificateChain
		}
	} else {
		panic("device certificate is a mandatory field")
	}
	//private key
	if ep_yaml.PrivateKey.PrivateKey1Path != "" {
		if _, err := os.Stat(ep_yaml.PrivateKey.PrivateKey1Path); errors.Is(err, os.ErrNotExist) {
			panic("private key path 1 does not exist")
		} else {
			endpoint_config.PrivateKey.Path = ep_yaml.PrivateKey.PrivateKey1Path
		}
	} else {
		panic("private key 1 is a mandatory field")
	}

	// if ep_yaml.Pin != "" {
	// 	endpoint_config.PKCS11.LongTermCryptoModule.Pin = ep_yaml.Pin
	// }
	//
	// if ep_yaml.SecureElementMiddlewarePath != "" {
	// 	endpoint_config.PKCS11.LongTermCryptoModule.Path = ep_yaml.SecureElementMiddlewarePath
	// } else {
	// 	log.Info().Msg("no smart card used. secure middleware path empty")
	// }
	//
	if ep_yaml.PrivateKey.PrivateKey2Path != "" {
		panic("second private key not supported yet. Please leave private key 2 empty")
	}
	//root cert
	if ep_yaml.RootCertificate != "" {
		if _, err := os.Stat(ep_yaml.RootCertificate); errors.Is(err, os.ErrNotExist) {
			panic("root cert path does not exist")
		} else {
			endpoint_config.RootCertificate.Path = ep_yaml.RootCertificate
		}
	} else {
		panic("root cert is not provided")
	}
	//keylogfile
	if ep_yaml.KeylogFile != "" {
		if _, err := os.Stat(ep_yaml.KeylogFile); errors.Is(err, os.ErrNotExist) {
			log.Warn().Msg("keylog file does not exist")
		} else {
			endpoint_config.KeylogFile = ep_yaml.KeylogFile
		}
	} else {
		log.Info().Msg("keylog_file not set")
	}

	return endpoint_config
}

func GetNodeServerConfig() NodeServerConfig {
	var ks_ep_config KS_EndpointConfig
	address := viper.GetString("node_server.address")
	address_http := viper.GetString("node_server.address_http")

	ks_ep_config.MutualAuthentication = viper.GetBool("node_server.endpoint_config.mutual_authentication")
	if !viper.IsSet("node_server.endpoint_config.mutual_authentication") {
		panic("mutual authentication is not in config.yaml")
	}

	ks_ep_config.NoEncryption = viper.GetBool("node_server.endpoint_config.no_encryption")
	if !viper.IsSet("node_server.endpoint_config.no_encryption") {
		panic("NoEncryption value not set")
	}

	ks_ep_config.ASLKeyExchangeMethod = viper.GetString("node_server.endpoint_config.key_exchange_method")
	if ks_ep_config.ASLKeyExchangeMethod == "" {
		panic("ASLKeyExchangeMethod is not set or empty")
	}
	ks_ep_config.HybridSignatureMode = viper.GetString("node_server.endpoint_config.hybrid_signature_mode")
	if ks_ep_config.HybridSignatureMode == "" {
		panic("Hybridsignaturemode is not set or empty")
	}

	ks_ep_config.SecureElementMiddlewarePath = viper.GetString("node_server.endpoint_config.secure_element_middleware_path")
	if ks_ep_config.SecureElementMiddlewarePath != "" {
		ks_ep_config.SecureElementMiddlewarePath = util.AbsolutePathFromConfigPath(ks_ep_config.SecureElementMiddlewarePath)
	}
	ks_ep_config.Pin = viper.GetString("node_server.endpoint_config.pin")

	//Certificates:
	ks_ep_config.PrivateKey.PrivateKey1Path = viper.GetString("node_server.endpoint_config.private_key.path_1")
	if ks_ep_config.PrivateKey.PrivateKey1Path == "" {
		panic("PrivateKey1Path is not set or empty")
	}
	if ks_ep_config.PrivateKey.PrivateKey1Path != "" {
		ks_ep_config.PrivateKey.PrivateKey1Path = util.AbsolutePathFromConfigPath(ks_ep_config.PrivateKey.PrivateKey1Path)
	}

	//server/device
	ks_ep_config.DeviceCertificateChain = viper.GetString("node_server.endpoint_config.server_cert_path")
	if ks_ep_config.DeviceCertificateChain == "" {
		panic("DeviceCertificateChain is not set or empty")
	}
	ks_ep_config.DeviceCertificateChain = util.AbsolutePathFromConfigPath(ks_ep_config.DeviceCertificateChain)
	//root cert

	ks_ep_config.RootCertificate = util.AbsolutePathFromConfigPath(viper.GetString("node_server.endpoint_config.root_certificate"))
	if ks_ep_config.RootCertificate == "" {
		panic("RootCertificate is not set or empty")
	}
	//keylogfile
	ks_ep_config.KeylogFile = util.AbsolutePathFromConfigPath(viper.GetString("node_server.endpoint_config.key_log_file"))
	if ks_ep_config.KeylogFile != "" {
		ks_ep_config.KeylogFile = util.AbsolutePathFromConfigPath(ks_ep_config.KeylogFile)
	}

	logLevelStr := viper.GetString("node_server.log.level")
	logLevel, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = zerolog.DebugLevel
	}

	return NodeServerConfig{
		Address:      address,
		AddressHTTP:  address_http,
		// ASL_Endpoint: parse_endpoint(&ks_ep_config),
		GinMode:      viper.GetString("node_server.log.mode"),

		Log: LogConfig{
			Level: logLevel,
		},
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

func GetLogConfig() LogConfig {
	logLevelStr := viper.GetString("log.level")
	logLevel, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = zerolog.DebugLevel
	}

	logFormatOpt := viper.GetString("log.format")
	var logFormat string
	switch logFormatOpt {
	case "json":
		logFormat = JSONLogFormat
	case "text":
		logFormat = TextLogFormat
	case "":
		logFormat = TextLogFormat
	default:
		log.Error().
			Str("func", "GetLogConfig").
			Msgf("Could not parse log format: %s. Valid choices are 'json' or 'text'", logFormatOpt)
	}

	return LogConfig{
		Format: logFormat,
		Level:  logLevel,
	}
}

func GetKritis3mScaleConfig() (*Config, error) {

	return &Config{

		Database:     GetDatabaseConfig(),
		Log_Database: GetLogDatabaseConfig(),
		NodeServer:   GetNodeServerConfig(),
		ACL:          GetACLConfig(),
		ASLConfig:    GetASLConfig(),

		// CLI: CLIConfig{
		// 	Address:  viper.GetString("cli.address"),
		// 	Timeout:  viper.GetDuration("cli.timeout"),
		// 	Insecure: viper.GetBool("cli.insecure"),
		// },

		Log: GetLogConfig(),
	}, nil
}

func GetLogDatabaseConfig() DatabaseConfig {

	debug := viper.GetBool("log_database.debug")

	type_ := viper.GetString("log_database.type")

	switch type_ {
	case "sqlite":
		type_ = "sqlite3"
	default:
		log.Fatal().
			Msgf("invalid database type %q, must be  sqlite3 ", type_)
	}

	return DatabaseConfig{
		Type:  type_,
		Debug: debug,
		Sqlite: SqliteConfig{
			Path: util.AbsolutePathFromConfigPath(
				viper.GetString("log_database.sqlite.path"),
			),
		},
	}

}
func GetDatabaseConfig() DatabaseConfig {
	debug := viper.GetBool("database.debug")

	type_ := viper.GetString("database.type")

	switch type_ {
	case "sqlite":
		type_ = "sqlite3"
	default:
		log.Fatal().
			Msgf("invalid database type %q, must be  sqlite3 ", type_)
	}

	logLevelStr := viper.GetString("database.log_level")
	logLevel, err := zerolog.ParseLevel(logLevelStr)
	if err != nil {
		logLevel = zerolog.DebugLevel
	}
	return DatabaseConfig{
		Type:  type_,
		Debug: debug,
		Level: logLevel,
		Sqlite: SqliteConfig{
			Path: util.AbsolutePathFromConfigPath(
				viper.GetString("database.sqlite.path"),
			),
		},
	}
}
