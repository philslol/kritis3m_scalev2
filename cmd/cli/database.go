package cli

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	log.Debug().Msg("Registering db_test command")
	rootCmd.AddCommand(db_test)
}

var db_test = &cobra.Command{
	Use:   "db_test",  // <-- Change from "scale" to "db_test"
	Short: "Run database test",
	Long:  "A CLI command to test database connection",
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug().Msg("starting kritis3m_scale app")

		// Start headscale app
		app, _ := getKritis3mScaleApp()
		app.Serve()
	},
}
