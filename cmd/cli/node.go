package cli

import (
	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	log.Debug().Msg("Registering node commands")
	rootCmd.AddCommand(nodeCli)

	// Create command flags
	createNodeCmd.Flags().StringP("serial-number", "s", "", "Serial number of the node")
	createNodeCmd.MarkFlagRequired("serial-number")

	createNodeCmd.Flags().Int32P("network-index", "n", 0, "Network index of the node")

	createNodeCmd.Flags().StringP("locality", "l", "", "Locality of the node")
	createNodeCmd.MarkFlagRequired("locality")

	createNodeCmd.Flags().StringP("version_number", "v", "", "Reference to the version")
	createNodeCmd.MarkFlagRequired("version_number")

	// Add all commands to node CLI
	nodeCli.AddCommand(createNodeCmd)

	readNodeCmd.Flags().StringP("serial-number", "s", "", "Serial number of the node")
	readNodeCmd.Flags().StringP("list", "l", "", "Serial number of the node")
	readNodeCmd.Flags().Int32P("id", "i", 0, "id of node")
	readNodeCmd.Flags().StringP("version_number", "v", "", "Reference to the version")

	nodeCli.AddCommand(readNodeCmd)

	nodeCli.AddCommand(updateNodeCmd)

	nodeCli.AddCommand(deleteNodeCmd)
}

var nodeCli = &cobra.Command{
	Use:   "node",
	Short: "Manage nodes",
	Long:  "Create, read, update, and delete nodes in the system",
}

var createNodeCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new node",
	Long:  "Create a new node with the specified parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		serialNumber, _ := cmd.Flags().GetString("serial-number")
		networkIndex, _ := cmd.Flags().GetInt32("network-index")
		locality, _ := cmd.Flags().GetString("locality")
		versionSetID, _ := cmd.Flags().GetString("version_number")
		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.CreateNodeRequest{
			SerialNumber: serialNumber,
			NetworkIndex: networkIndex,
			Locality:     &locality,
			VersionSetId: versionSetID,
		}

		rsp, err := client.CreateNode(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create node")
		}

		//log all fields
		log.Info().Msgf("Node created: %v", rsp)

		return nil
	},
}

var readNodeCmd = &cobra.Command{
	Use:   "read",
	Short: "Read node details",
	Long:  "Read and display details of a specific node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serialNumber := args[0]

		// TODO: Implement node read logic
		log.Info().
			Str("serial-number", serialNumber).
			Msg("Reading node details")

		return nil
	},
}

var updateNodeCmd = &cobra.Command{
	Use:   "update [serial-number]",
	Short: "Update node details",
	Long:  "Update the details of an existing node",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serialNumber := args[0]

		// TODO: Implement node update logic
		log.Info().
			Str("serial-number", serialNumber).
			Msg("Updating node details")

		return nil
	},
}

var deleteNodeCmd = &cobra.Command{
	Use:   "delete [serial-number]",
	Short: "Delete a node",
	Long:  "Delete an existing node from the system",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serialNumber := args[0]

		// TODO: Implement node deletion logic
		log.Info().
			Str("serial-number", serialNumber).
			Msg("Deleting node")

		return nil
	},
}
