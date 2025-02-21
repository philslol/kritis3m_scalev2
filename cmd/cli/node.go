package cli

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

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

	// Read command flags
	readNodeCmd.Flags().Int32P("id", "i", 0, "ID of the node")
	readNodeCmd.Flags().StringP("version-set-id", "v", "", "Version set ID to list nodes for")
	readNodeCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	nodeCli.AddCommand(readNodeCmd)

	// Update command flags
	updateNodeCmd.Flags().Int32P("id", "i", 0, "ID of the node")
	updateNodeCmd.MarkFlagRequired("id")
	updateNodeCmd.Flags().StringP("serial-number", "s", "", "Serial number of the node")
	updateNodeCmd.Flags().Int32P("network-index", "n", 0, "Network index of the node")
	updateNodeCmd.Flags().StringP("locality", "l", "", "Locality of the node")
	updateNodeCmd.Flags().StringP("version_number", "v", "", "Reference to the version")
	nodeCli.AddCommand(updateNodeCmd)

	// Delete command flags
	deleteNodeCmd.Flags().Int32P("id", "i", 0, "ID of the node")
	deleteNodeCmd.MarkFlagRequired("id")
	nodeCli.AddCommand(deleteNodeCmd)

	// List command flags
	listNodesCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	nodeCli.AddCommand(listNodesCmd)
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
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		versionSetId, _ := cmd.Flags().GetString("version-set-id")

		if id == 0 && versionSetId == "" {
			return fmt.Errorf("either id or version-set-id must be provided")
		}

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		if id != 0 {
			request := &v1.GetNodeRequest{
				Id:           id,
				VersionSetId: versionSetId,
			}

			rsp, err := client.GetNode(ctx, request)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get node")
			}

			if HasMachineOutputFlag() {
				SuccessOutput(rsp, "", outputFormat)
				return nil
			}

			PrintNodeResponseAsTable([]*v1.NodeResponse{rsp})
			return nil
		}

		// If version-set-id is provided, list nodes for that version set
		request := &v1.ListNodesRequest{
			VersionSetId: &versionSetId,
		}

		rsp, err := client.ListNodes(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list nodes")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.GetNodes(), "", outputFormat)
			return nil
		}

		PrintNodeResponseAsTable(rsp.GetNodes())
		return nil
	},
}

var updateNodeCmd = &cobra.Command{
	Use:   "update",
	Short: "Update node details",
	Long:  "Update the details of an existing node",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
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

		request := &v1.UpdateNodeRequest{
			Id:           id,
			SerialNumber: &serialNumber,
			NetworkIndex: &networkIndex,
			Locality:     &locality,
			VersionSetId: &versionSetID,
		}

		_, err = client.UpdateNode(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to update node")
		}

		log.Info().Msg("Node updated successfully")
		return nil
	},
}

var deleteNodeCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a node",
	Long:  "Delete an existing node from the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.DeleteNodeRequest{
			Id: id,
		}

		_, err = client.DeleteNode(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to delete node")
		}

		log.Info().Msg("Node deleted successfully")
		return nil
	},
}

var listNodesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all nodes",
	Long:  "List all nodes in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.ListNodesRequest{}

		rsp, err := client.ListNodes(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list nodes")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.GetNodes(), "", outputFormat)
			return nil
		}

		PrintNodeResponseAsTable(rsp.GetNodes())
		return nil
	},
}

func PrintNodeResponseAsTable(nodes []*v1.NodeResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tSERIAL NUMBER\tNETWORK INDEX\tLOCALITY\tVERSION SET ID\tLAST SEEN")

	for _, nodeResp := range nodes {
		node := nodeResp.GetNode()
		fmt.Fprintf(w, "%d\t%s\t%d\t%s\t%s\t%s\n",
			node.Id,
			node.SerialNumber,
			node.NetworkIndex,
			node.Locality,
			node.VersionSetId,
			node.LastSeen.AsTime().Format(time.RFC3339),
		)
	}
	w.Flush()
}
