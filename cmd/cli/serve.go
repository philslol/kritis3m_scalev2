package cli

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	log.Debug().Msg("Starting Serve cmd")
	rootCmd.AddCommand(serveCmd)

}

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Starts the server",
	Long:  "Starts kritis3m_scale server",
	RunE: func(cmd *cobra.Command, args []string) error {
		scale, err := getKritis3mScaleApp()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get kritis3m_scale app")
		}
		scale.Serve()
		return nil
	},
}
