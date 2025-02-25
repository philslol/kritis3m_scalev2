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

	createNodeCmd.Flags().StringP("version-number", "v", "", "Reference to the version")
	createNodeCmd.MarkFlagRequired("version-number")

	createNodeCmd.Flags().StringP("created-by", "u", "", "User creating the node")
	createNodeCmd.MarkFlagRequired("created-by")

	// Add all commands to node CLI
	nodeCli.AddCommand(createNodeCmd)

	// Read command flags
	readNodeCmd.Flags().Int32P("id", "i", 0, "ID of node")

	readNodeCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	readNodeCmd.MarkFlagRequired("version-number")
	readNodeCmd.Flags().StringP("serial-number", "s", "", "Serial number of the node")
	readNodeCmd.MarkFlagRequired("serial-number")

	readNodeCmd.Flags().Bool("include", false, "Include related configs")
	readNodeCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	nodeCli.AddCommand(readNodeCmd)

	// Update command flags
	updateNodeCmd.Flags().Int32P("id", "i", 0, "ID of the node")

	updateNodeCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	updateNodeCmd.MarkFlagRequired("version-number")
	updateNodeCmd.Flags().StringP("serial-number", "s", "", "Serial number of the node")
	updateNodeCmd.MarkFlagRequired("serial-number")
	updateNodeCmd.Flags().Int32P("network-index", "n", 0, "Network index of the node")
	updateNodeCmd.Flags().StringP("locality", "l", "", "Locality of the node")
	nodeCli.AddCommand(updateNodeCmd)

	// Delete command flags
	deleteNodeCmd.Flags().Int32P("id", "i", 0, "ID of the node")
	deleteNodeCmd.MarkFlagRequired("id")
	nodeCli.AddCommand(deleteNodeCmd)

	// List command flags
	listNodesCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	listNodesCmd.Flags().Bool("include", false, "Include related configs")
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
		versionSetID, _ := cmd.Flags().GetString("version-number")
		createdBy, _ := cmd.Flags().GetString("created-by")
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
			User:         createdBy,
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

		versionSetID, _ := cmd.Flags().GetString("version-number")
		serialNumber, _ := cmd.Flags().GetString("serial-number")
		include, _ := cmd.Flags().GetBool("include")

		if versionSetID == "" && serialNumber == "" && id == 0 {
			return fmt.Errorf("either version-number or serial-number or id must be provided")
		}

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		var request *v1.GetNodeRequest

		if id != 0 {
			request = &v1.GetNodeRequest{
				Query: &v1.GetNodeRequest_Id{
					Id: id,
				},
				Include: &include,
			}
		} else {
			request = &v1.GetNodeRequest{
				Query: &v1.GetNodeRequest_NodeQuery{
					NodeQuery: &v1.NodeNameQuery{
						SerialNumber: serialNumber,
						VersionSetId: versionSetID,
					},
				},
				Include: &include,
			}
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
	},
}

var updateNodeCmd = &cobra.Command{
	Use:   "update",
	Short: "Update node details",
	Long:  "Update the details of an existing node",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetID, _ := cmd.Flags().GetString("version-number")
		serialNumber, _ := cmd.Flags().GetString("serial-number")
		networkIndex, _ := cmd.Flags().GetInt32("network-index")
		locality, _ := cmd.Flags().GetString("locality")
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		var request *v1.UpdateNodeRequest

		if id != 0 {
			request = &v1.UpdateNodeRequest{
				Query: &v1.UpdateNodeRequest_Id{
					Id: id,
				},
			}
		} else {
			request = &v1.UpdateNodeRequest{
				Query: &v1.UpdateNodeRequest_NodeQuery{
					NodeQuery: &v1.NodeNameQuery{
						VersionSetId: versionSetID,
						SerialNumber: serialNumber,
					},
				},
			}
		}

		request.NetworkIndex = &networkIndex
		request.Locality = &locality

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
		serialNumber, _ := cmd.Flags().GetString("serial-number")
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.DeleteNodeRequest{
			SerialNumber: serialNumber,
			VersionSetId: versionSetID,
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
		versionSetID, _ := cmd.Flags().GetString("version-number")
		include, _ := cmd.Flags().GetBool("include")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.ListNodesRequest{
			VersionSetId: &versionSetID,
			Include:      &include,
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
