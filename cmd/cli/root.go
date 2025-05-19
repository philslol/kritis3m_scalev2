package cli

import (
	"fmt"
	"os"

	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	deprecateNamespaceMessage = "use --user"
)

var cli_logger zerolog.Logger

var outputFormat string

var cfgFile string = ""

func init() {
	if len(os.Args) > 1 &&
		(os.Args[1] == "version" || os.Args[1] == "mockoidc" || os.Args[1] == "completion") {
		return
	}

	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().
		StringVar(&cfgFile, "config", "./server/config.yaml", "Path to the configuration file (default is ./server/config.yaml)")
	rootCmd.PersistentFlags().
		StringP("output", "o", "", "Output format. Empty for human-readable, 'json', 'json-line' or 'yaml'")
	rootCmd.PersistentFlags().
		Bool("force", false, "Disable prompts and forces the execution")
}

func initConfig() {
	// If the --config flag is not set, use the default value from the flag definition
	if cfgFile == "" {
		cfgFile = "./server/config.yaml"
	} else {
		cli_logger.Debug().Msgf("using config file %s", cfgFile)
	}

	// Attempt to load the configuration
	err := types.LoadConfig(cfgFile, true)
	if err != nil {
		cli_logger.Fatal().Err(err).Msgf("Error loading config file %s", cfgFile)
	}
	cfg, err := types.GetKritis3mScaleConfig()
	if err != nil {
		cli_logger.Fatal().Caller().Err(err).Msg("Failed to get kritis3m-scale configuration")
	}

	cli_logger = types.CreateLogger("cli", cfg.CLILog.Level, cfg.CLILog.File)

	machineOutput := HasMachineOutputFlag()

	zerolog.SetGlobalLevel(cfg.Log.Level)
	cli_logger.Debug().Msgf("logger with global level %s", zerolog.FormattedLevels[zerolog.GlobalLevel()])

	// If the user has requested a "node" readable format,
	// then disable login so the output remains valid.
	if machineOutput {
		cli_logger.Debug().Msg("machine output. Disabled logging")
		zerolog.SetGlobalLevel(zerolog.Disabled)
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
	cli_logger.Debug().Msg("exit")
}

// GetRootCommand returns the root command for testing purposes
func GetRootCommand() *cobra.Command {
	initConfig()
	return rootCmd
}
