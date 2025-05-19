package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	cli_logger.Debug().Msg("Starting Serve cmd")
	rootCmd.AddCommand(serveCmd)

}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Long:  "Starts kritis3m_scale server",
	RunE: func(cmd *cobra.Command, args []string) error {
		scale, err := getKritis3mScaleApp()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get kritis3m_scale app")
		}
		scale.Serve()
		return nil
	},
}
