package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/philslol/kritis3m_scalev2/control"
	"github.com/philslol/kritis3m_scalev2/control/types"
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

const (
	HeadscaleDateTimeFormat = "2006-01-02 15:04:05"
	SocketWritePermissions  = 0o666
)

func getClient() (context.Context, v1.SouthboundClient, *grpc.ClientConn, context.CancelFunc, error) {
	cfg, err := types.GetKritis3mScaleConfig()

	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to load configuration while creating headscale instance: %w", err)
	}

	log.Debug().
		Dur("timeout", cfg.CliConfig.Timeout).
		Msgf("Setting timeout")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.CliConfig.Timeout)

	grpcOptions := []grpc.DialOption{}
	//use insecure.NewCredentials()
	grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))

	address := cfg.CliConfig.ServerAddr
	log.Trace().Caller().Str("address", address).Msg("Connecting via gRPC")
	//grpc is DialContext is deprecated
	conn, err := grpc.NewClient(address, grpcOptions...)
	if err != nil {
		log.Fatal().Caller().Err(err).Msgf("Could not connect: %v", err)
		os.Exit(-1) // we get here if logging is suppressed (i.e., json output)
	}

	client := v1.NewSouthboundClient(conn)

	return ctx, client, conn, cancel, nil
}

func getKritis3mScaleApp() (*control.Kritis3m_Scale, error) {
	cfg, err := types.GetKritis3mScaleConfig()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load configuration while creating headscale instance: %w",
			err,
		)
	}

	app, err := control.NewKritis3m_scale(cfg)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func ErrorOutput(errResult error, override string, outputFormat string) {
	type errOutput struct {
		Error string `json:"error"`
	}

	SuccessOutput(errOutput{errResult.Error()}, override, outputFormat)
}

func HasMachineOutputFlag() bool {
	for _, arg := range os.Args {
		if arg == "json" || arg == "json-line" || arg == "yaml" {
			return true
		}
	}

	return false
}

func SuccessOutput(result interface{}, override string, outputFormat string) {
	var jsonBytes []byte
	var err error
	switch outputFormat {
	case "json":
		jsonBytes, err = json.MarshalIndent(result, "", "\t")
		if err != nil {
			log.Fatal().Err(err).Msg("failed to unmarshal output")
		}
	case "json-line":
		jsonBytes, err = json.Marshal(result)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to unmarshal output")
		}
	case "yaml":
		jsonBytes, err = yaml.Marshal(result)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to unmarshal output")
		}
	default:
		//nolint
		fmt.Println(override)

		return
	}

	//nolint
	fmt.Println(string(jsonBytes))
}
