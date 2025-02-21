package cli

import (
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	log.Debug().Msg("Registering version set commands")
	rootCmd.AddCommand(versionSetCli)

	// Create command flags
	createVersionSetCmd.Flags().StringP("name", "n", "", "Name of the version set")
	createVersionSetCmd.MarkFlagRequired("name")
	createVersionSetCmd.Flags().StringP("description", "d", "", "Description of the version set")
	createVersionSetCmd.Flags().StringP("created-by", "u", "", "User creating the version set")
	createVersionSetCmd.MarkFlagRequired("created-by")
	versionSetCli.AddCommand(createVersionSetCmd)

	readVersionSetCmd.Flags().StringP("id", "i", "", "ID of the version set")
	readVersionSetCmd.MarkFlagRequired("id")
	versionSetCli.AddCommand(readVersionSetCmd)

	updateVersionSetCmd.Flags().StringP("id", "i", "", "ID of the version set")
	updateVersionSetCmd.MarkFlagRequired("id")
	updateVersionSetCmd.Flags().StringP("name", "n", "", "New name for the version set")
	updateVersionSetCmd.Flags().StringP("description", "d", "", "New description for the version set")
	versionSetCli.AddCommand(updateVersionSetCmd)

	deleteVersionSetCmd.Flags().StringP("id", "i", "", "ID of the version set")
	deleteVersionSetCmd.MarkFlagRequired("id")
	versionSetCli.AddCommand(deleteVersionSetCmd)
}

var versionSetCli = &cobra.Command{
	Use:   "version-set",
	Short: "Manage version sets",
	Long:  "Create, read, update, and delete version sets in the system",
}

var createVersionSetCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new version set",
	Long:  "Create a new version set with the specified parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")
		createdBy, _ := cmd.Flags().GetString("created-by")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.CreateVersionSetRequest{
			Name:        name,
			Description: description,
			CreatedBy:   createdBy,
		}

		rsp, err := client.CreateVersionSet(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create version set")
		}

		log.Info().Msgf("Version set created: %v", rsp)
		return nil
	},
}

var readVersionSetCmd = &cobra.Command{
	Use:   "read",
	Short: "Read version set details",
	Long:  "Read and display details of a specific version set",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.GetVersionSetRequest{
			Id: id,
		}

		rsp, err := client.GetVersionSet(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to read version set")
		}

		log.Info().Msgf("Version set: %v", rsp.GetVersionSet())
		return nil
	},
}

var updateVersionSetCmd = &cobra.Command{
	Use:   "update",
	Short: "Update version set details",
	Long:  "Update the details of an existing version set",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")
		name, _ := cmd.Flags().GetString("name")
		description, _ := cmd.Flags().GetString("description")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.UpdateVersionSetRequest{
			Id:          id,
			Name:        name,
			Description: description,
		}

		rsp, err := client.UpdateVersionSet(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to update version set")
		}

		log.Info().Msgf("Version set updated: %v", rsp)
		return nil
	},
}

var deleteVersionSetCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a version set",
	Long:  "Delete an existing version set from the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetString("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.DeleteVersionSetRequest{
			Id: id,
		}

		rsp, err := client.DeleteVersionSet(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to delete version set")
		}

		log.Info().Msgf("Version set deleted: %v", rsp)
		return nil
	},
}
