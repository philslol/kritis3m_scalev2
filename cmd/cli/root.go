package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	deprecateNamespaceMessage = "use --user"
)

var cfgFile string = "./config.yaml"

func init() {
	if len(os.Args) > 1 &&
		(os.Args[1] == "version" || os.Args[1] == "mockoidc" || os.Args[1] == "completion") {
		return
	}

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "", "Path to the configuration file (default is ./config.yaml)")
	rootCmd.PersistentFlags().
		StringP("output", "o", "", "Output format. Empty for human-readable, 'json', 'json-line' or 'yaml'")
	rootCmd.PersistentFlags().
		Bool("force", false, "Disable prompts and forces the execution")
}

func initConfig() {
	// If the --config flag is not set, use a default value
	if cfgFile == "" {
		cfgFile = "./config.yaml" // Default file path if no argument provided
	}
	log.Debug().Msg("in function initconfig root")

	// Convert the path to absolute for clarity
	cfgFile, _ = filepath.Abs(cfgFile)

	// Attempt to load the configuration
	err := types.LoadConfig(cfgFile, true)
	if err != nil {
		log.Fatal().Err(err).Msgf("Error loading config file %s", cfgFile)
	}
	cfg, err := types.GetKritis3mScaleConfig()
	if err != nil {
		log.Fatal().Caller().Err(err).Msg("Failed to get kritis3m-scale configuration")
	}

	machineOutput := HasMachineOutputFlag()

	zerolog.SetGlobalLevel(cfg.Log.Level)

	// If the user has requested a "node" readable format,
	// then disable login so the output remains valid.
	if machineOutput {
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	if cfg.Log.Format == types.JSONLogFormat {
		log.Logger = log.Output(os.Stdout)
	}

}

var rootCmd = &cobra.Command{
	Use:   "kritis3m_scale",
	Short: "kritis3m_scale - a kritis3m control server",
	Long: `
krits3m_scale is a server that is used to control kritis3m gateways
	github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_scale`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
		return
	}

	log.Debug().Msg("exit")
}
