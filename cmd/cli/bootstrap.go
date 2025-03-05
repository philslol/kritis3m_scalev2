package cli

import (
	"time"

	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(bootstrapCmd)
	rootCmd.AddCommand(resetCmd)

}

var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the database",
	Long:  "Clear all tables in the database",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := getKritis3mScaleApp()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create kritis3m-scale instance")
		}

		sm, _, cancel, err := app.GetRawDB(10 * time.Second)
		defer cancel()

		err = sm.ResetDatabase()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to reset database")
		}

		log.Info().Msg("Database reset successfully")
		return nil
	},
}

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap the database",
	Long:  "Initialize database schema and create default version set",
	RunE: func(cmd *cobra.Command, args []string) error {

		app, err := getKritis3mScaleApp()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create kritis3m-scale instance")
		}

		// Initialize database schema
		// ctx with timeout
		sm, ctx, cancel, err := app.GetRawDB(10 * time.Second)
		defer cancel()

		err = sm.InitializeSchema()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to initialize database schema")
		}

		log.Info().Msg("Creating initial version set")

		defaultVersionSet := types.VersionSet{
			Name:        "initial version set",
			Description: stringPtr("On startup created version set"),
			State:       types.VERSION_STATE_PENDING_DEPLOYMENT,
			CreatedBy:   "admin",
		}
		id, err := sm.CreateVersionSet(ctx, defaultVersionSet)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create initial version set")
		}
		log.Info().Msgf("Created initial version set with ID: %s", id)
		defaultVersionSet.ID = id

		// Create default version transition
		defaultTransition := &types.VersionTransition{
			ToVersionSetID: defaultVersionSet.ID,
			Status:         types.VersionTransitionActive,
			CreatedBy:      "admin",
		}
		err = sm.CreateVersionTransition(ctx, defaultTransition)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create default version transition")
		}
		log.Info().Msg("Created default version transition")

		return nil
	},
}

func stringPtr(s string) *string {
	return &s
}
