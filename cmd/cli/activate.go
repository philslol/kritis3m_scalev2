package cli

import (
	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var activateCmd = &cobra.Command{
	Use:   "activate",
	Short: "activate nodes",
	Long:  "activate a node in the network",
}

func init() {
	log.Debug().Msg("Registering proxy commands")

	rootCmd.AddCommand(activateCmd)

	activateNodeCmd.Flags().StringP("node-serial", "n", "", "Node serial number")
	activateNodeCmd.Flags().StringP("version-number", "v", "", "version number")

	activateNodeCmd.MarkFlagRequired("version-number")
	activateNodeCmd.MarkFlagRequired("node-serial")
	activateCmd.AddCommand(activateNodeCmd)

	activateGroupCmd.Flags().StringP("version-number", "v", "", "version number")
	activateGroupCmd.Flags().StringP("group", "g", "", "group name")

	activateGroupCmd.MarkFlagRequired("version-number")
	activateGroupCmd.MarkFlagRequired("group")
	activateCmd.AddCommand(activateGroupCmd)

	activateFleetCmd.Flags().StringP("version-number", "v", "", "version number")
	activateFleetCmd.MarkFlagRequired("version-number")
	activateCmd.AddCommand(activateFleetCmd)

	// Create command flags
}

var activateNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "activate a node",
	Long:  "activate a node in the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeSerial, _ := cmd.Flags().GetString("node-serial")
		versionSetId, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		req := &grpc_southbound.ActivateNodeRequest{
			SerialNumber: nodeSerial,
			VersionSetId: versionSetId,
		}

		resp, err := client.ActivateNode(ctx, req)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to activate node")
		}
		log.Info().Msgf("Returncode is %d, with metadata %v", resp.Retcode, resp.Metadata)

		return nil
	},
}

var activateGroupCmd = &cobra.Command{
	Use:   "node",
	Short: "activate a group",
	Long:  "activate a group in the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		group, _ := cmd.Flags().GetString("group")
		versionSetId, _ := cmd.Flags().GetString("version-number")

		if versionSetId == "" || group == "" {
			log.Fatal().Msg("Version number or group name is missing")
		}

		req := &grpc_southbound.ActivateFleetRequest{
			VersionSetId: versionSetId,
			GroupName:    &group,
		}

		//get client
		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		resp, err := client.ActivateFleet(ctx, req)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to activate fleet")
		}
		log.Info().Msgf("Returncode is %d, with metadata %v", resp.Retcode, resp.Metadata)

		return nil
	},
}

var activateFleetCmd = &cobra.Command{
	Use:   "node",
	Short: "activate a fleet",
	Long:  "activate a fleet in the network",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetId, _ := cmd.Flags().GetString("version-number")

		if versionSetId == "" {
			log.Fatal().Msg("Version number is missing")
		}

		req := &grpc_southbound.ActivateFleetRequest{
			VersionSetId: versionSetId,
		}

		//get client
		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		resp, err := client.ActivateFleet(ctx, req)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to activate fleet")
		}
		log.Info().Msgf("Returncode is %d, with metadata %v", resp.Retcode, resp.Metadata)

		return nil
	},
}
