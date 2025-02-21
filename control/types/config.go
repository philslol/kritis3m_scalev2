package types

import (
	"errors"
	"fmt"
	"strings"
	"time"

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
	Log LogConfig

	Database DatabaseConfig

	Log_Database DatabaseConfig
	// CLI          CLIConfig
	ACL       ACLConfig
	ASLConfig asl.ASLConfig

	Broker       BrokerConfig
	ControlPlane ControlPlaneConfig

	CliConfig CliConfig
}

type ACLConfig struct {
	PolicyPath string
}

type BrokerConfig struct {
	Adress         string
	Log            LogConfig
	EndpointConfig asl.EndpointConfig
}

type ControlPlaneConfig struct {
	Adress         string
	Log            LogConfig
	EndpointConfig asl.EndpointConfig
}

type LogConfig struct {
	Format string
	Level  zerolog.Level
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
	return log_config
}

func parse_ASLEndpointConfig(basepath string) (*asl.EndpointConfig, error) {
	var kex asl.ASLKeyExchangeMethod
	var pkcs11 asl.PKCS11ASL
	var root asl.RootCertificate
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
	device.Path = viper.GetString(fmt.Sprintf("%s.%s", basepath, "device_cert"))
	root.Path = viper.GetString(fmt.Sprintf("%s.%s", basepath, "root_cert"))

	if root.Path == "" || device.Path == "" || private.Path == "" {
		err := fmt.Errorf("either device or root or private key is empty")
		log.Err(err)
		return nil, err
	}

	var ep_config asl.EndpointConfig = asl.EndpointConfig{
		KeylogFile:             viper.GetString(fmt.Sprintf("%s.%s", basepath, "key_log_file")),
		MutualAuthentication:   viper.GetBool(fmt.Sprintf("%s.%s", basepath, "mutual_authentication")),
		NoEncryption:           viper.GetBool(fmt.Sprintf("%s.%s", basepath, "no_encryption")),
		ASLKeyExchangeMethod:   kex,
		PKCS11:                 pkcs11,
		RootCertificate:        root,
		DeviceCertificateChain: device,
		PrivateKey:             private,
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

	control_plane_config.Log = log
	control_plane_config.EndpointConfig = *ep
	control_plane_config.Adress = viper.GetString("control_plane_config.server_address")
	if control_plane_config.Adress == "" {
		return nil, fmt.Errorf("no address specified for control plane address")
	}

	return &control_plane_config, nil
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
	return &Config{
		ACL:          GetACLConfig(),
		ASLConfig:    GetASLConfig(),
		Database:     *database_config,
		Broker:       *broker,
		ControlPlane: *ctrl_plane_cfg,
		Log:          parse_Log(""),
		CliConfig:    GetCliConfig(),
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
