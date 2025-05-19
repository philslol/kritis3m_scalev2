package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"github.com/spf13/cobra"
)

func init() {
	cli_logger.Debug().Msg("Registering endpoint commands")
	rootCmd.AddCommand(endpointCli)

	// Create command flags
	createEndpointCmd.Flags().StringP("name", "n", "", "Name of the endpoint")
	createEndpointCmd.MarkFlagRequired("name")
	createEndpointCmd.Flags().BoolP("mutual-auth", "m", false, "Enable mutual authentication")
	createEndpointCmd.Flags().BoolP("no-encryption", "e", false, "Disable encryption")
	createEndpointCmd.Flags().StringP("kex-method", "k", "ASL_KEX_DEFAULT", "ASL key exchange method")
	createEndpointCmd.Flags().StringP("cipher", "c", "", "Cipher configuration")
	createEndpointCmd.Flags().StringP("created-by", "u", "", "User creating the endpoint")
	createEndpointCmd.Flags().StringP("version-number", "v", "", "Reference to the version")
	endpointCli.AddCommand(createEndpointCmd)

	readEndpointCmd.Flags().Int32P("id", "i", 0, "ID of the endpoint (deprecated)")
	readEndpointCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	readEndpointCmd.MarkFlagRequired("version-number")
	readEndpointCmd.Flags().StringP("name", "n", "", "Name of the endpoint")
	readEndpointCmd.MarkFlagRequired("name")
	endpointCli.AddCommand(readEndpointCmd)

	updateEndpointCmd.Flags().Int32P("id", "i", 0, "ID of the endpoint (deprecated)")
	updateEndpointCmd.Flags().StringP("name", "n", "", "Name of the endpoint")
	updateEndpointCmd.MarkFlagRequired("name")
	updateEndpointCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	updateEndpointCmd.MarkFlagRequired("version-number")
	updateEndpointCmd.Flags().BoolP("mutual-auth", "m", false, "Enable mutual authentication")
	updateEndpointCmd.Flags().BoolP("no-encryption", "e", false, "Disable encryption")
	updateEndpointCmd.Flags().StringP("kex-method", "k", "", "ASL key exchange method")
	updateEndpointCmd.Flags().StringP("cipher", "c", "", "Cipher configuration")
	endpointCli.AddCommand(updateEndpointCmd)

	deleteEndpointCmd.Flags().Int32P("id", "i", 0, "ID of the endpoint")
	deleteEndpointCmd.MarkFlagRequired("id")
	endpointCli.AddCommand(deleteEndpointCmd)

	listEndpointsCmd.Flags().StringP("version-number", "v", "", "Reference to the version")
	endpointCli.AddCommand(listEndpointsCmd)

	// Add output format flags
	readEndpointCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	listEndpointsCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
}

var endpointCli = &cobra.Command{
	Use:   "endpoint",
	Short: "Manage endpoints",
	Long:  "Create, read, update, and delete endpoints in the system",
}

var createEndpointCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new endpoint",
	Long:  "Create a new endpoint with the specified parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		mutualAuth, _ := cmd.Flags().GetBool("mutual-auth")
		noEncryption, _ := cmd.Flags().GetBool("no-encryption")
		kexMethod, _ := cmd.Flags().GetString("kex-method")
		cipher, _ := cmd.Flags().GetString("cipher")
		createdBy, _ := cmd.Flags().GetString("created-by")
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.CreateEndpointConfigRequest{
			Name:                 name,
			MutualAuth:           mutualAuth,
			NoEncryption:         noEncryption,
			AslKeyExchangeMethod: types.ASLKeyExchangeMethodToProto(kexMethod),
			Cipher:               &cipher,
			CreatedBy:            createdBy,
			VersionSetId:         versionSetID,
		}

		rsp, err := client.CreateEndpointConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to create endpoint")
		}

		cli_logger.Info().Msgf("Endpoint created: %v", rsp)
		return nil
	},
}

var readEndpointCmd = &cobra.Command{
	Use:   "read",
	Short: "Read endpoint details",
	Long:  "Read and display details of a specific endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		versionSetID, _ := cmd.Flags().GetString("version-number")
		name, _ := cmd.Flags().GetString("name")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.GetEndpointConfigRequest{}
		if id != 0 {
			request.Query = &grpc_southbound.GetEndpointConfigRequest_Id{Id: id}
		} else if versionSetID != "" && name != "" {
			request = &grpc_southbound.GetEndpointConfigRequest{
				Query: &grpc_southbound.GetEndpointConfigRequest_EndpointConfigQuery{
					EndpointConfigQuery: &grpc_southbound.EndpointConfigNameQuery{
						VersionSetId: versionSetID,
						Name:         name,
					},
				},
			}
		} else {
			cli_logger.Fatal().Msg("Must specify either id or version-number and name")
		}

		rsp, err := client.GetEndpointConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get endpoint")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp, "", outputFormat)
			return nil
		}

		PrintEndpointsAsTable([]*grpc_southbound.EndpointConfig{rsp})
		return nil
	},
}

var updateEndpointCmd = &cobra.Command{
	Use:   "update",
	Short: "Update endpoint details",
	Long:  "Update the details of an existing endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		name, _ := cmd.Flags().GetString("name")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		cli_logger.Debug().Msgf("versionSetID: %s, name: %s", versionSetID, name)

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.UpdateEndpointConfigRequest{}

		// Set the query based on id or version+name
		if id != 0 {
			request.Query = &grpc_southbound.UpdateEndpointConfigRequest_Id{Id: id}
		} else if versionSetID != "" && name != "" {
			request.Query = &grpc_southbound.UpdateEndpointConfigRequest_EndpointConfigQuery{
				EndpointConfigQuery: &grpc_southbound.EndpointConfigNameQuery{
					VersionSetId: versionSetID,
					Name:         name,
				},
			}
		} else {
			cli_logger.Fatal().Msg("Must specify either id or version-number and name")
		}

		// Only add fields to the request if they were expliceitly set by user
		if cmd.Flags().Changed("name") {
			request.Name = &name
		}

		if cmd.Flags().Changed("mutual-auth") {
			mutualAuth, _ := cmd.Flags().GetBool("mutual-auth")
			request.MutualAuth = &mutualAuth
		}

		if cmd.Flags().Changed("no-encryption") {
			noEncryption, _ := cmd.Flags().GetBool("no-encryption")
			request.NoEncryption = &noEncryption
		}

		if cmd.Flags().Changed("kex-method") {
			kexMethod, _ := cmd.Flags().GetString("kex-method")
			kexMethodEnum := types.ASLKeyExchangeMethodToProto(kexMethod)
			request.AslKeyExchangeMethod = &kexMethodEnum
		}

		if cmd.Flags().Changed("cipher") {
			cipher, _ := cmd.Flags().GetString("cipher")
			request.Cipher = &cipher
		}

		_, err = client.UpdateEndpointConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to update endpoint")
		}

		cli_logger.Info().Msgf("Endpoint updated successfully")
		return nil
	},
}

var deleteEndpointCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an endpoint",
	Long:  "Delete an existing endpoint from the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.DeleteEndpointConfigRequest{
			Id: id,
		}

		_, err = client.DeleteEndpointConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to delete endpoint")
		}

		cli_logger.Info().Msgf("Endpoint deleted successfully")
		return nil
	},
}

var listEndpointsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all endpoints",
	Long:  "List all endpoints in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.ListEndpointConfigsRequest{}

		if versionSetID != "" {
			request.VersionSetId = &versionSetID
		}
		rsp, err := client.ListEndpointConfigs(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to list endpoints")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.Configs, "", outputFormat)
			return nil
		}

		PrintEndpointsAsTable(rsp.Configs)
		return nil
	},
}

func PrintEndpointsAsTable(endpoints []*grpc_southbound.EndpointConfig) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Version Set ID\t\t\tID\tNAME\tMUTUAL AUTH\tNO ENCRYPTION\tKEX METHOD\tCIPHER\tCREATED BY")

	for _, ep := range endpoints {
		fmt.Fprintf(w, "%s\t\t\t%d\t%s\t%t\t%t\t%s\t%s\t%s\n",
			ep.VersionSetId,
			ep.Id,
			ep.Name,
			ep.MutualAuth,
			ep.NoEncryption,
			ep.AslKeyExchangeMethod,
			*ep.Cipher,
			ep.CreatedBy,
		)
	}
	w.Flush()
}
