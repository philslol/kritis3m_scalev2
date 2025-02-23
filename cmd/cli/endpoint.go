package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func init() {
	log.Debug().Msg("Registering endpoint commands")
	rootCmd.AddCommand(endpointCli)

	// Create command flags
	createEndpointCmd.Flags().StringP("name", "n", "", "Name of the endpoint")
	createEndpointCmd.MarkFlagRequired("name")
	createEndpointCmd.Flags().BoolP("mutual-auth", "m", false, "Enable mutual authentication")
	createEndpointCmd.Flags().BoolP("no-encryption", "e", false, "Disable encryption")
	createEndpointCmd.Flags().StringP("kex-method", "k", "ASL_KEX_DEFAULT", "ASL key exchange method")
	createEndpointCmd.Flags().StringP("cipher", "c", "", "Cipher configuration")
	createEndpointCmd.Flags().StringP("created-by", "u", "", "User creating the endpoint")
	createEndpointCmd.Flags().StringP("version_number", "v", "", "Reference to the version")
	endpointCli.AddCommand(createEndpointCmd)

	readEndpointCmd.Flags().Int32P("id", "i", 0, "ID of the endpoint")
	readEndpointCmd.MarkFlagRequired("id")
	endpointCli.AddCommand(readEndpointCmd)

	updateEndpointCmd.Flags().Int32P("id", "i", 0, "ID of the endpoint")
	updateEndpointCmd.MarkFlagRequired("id")
	updateEndpointCmd.Flags().StringP("name", "n", "", "New name for the endpoint")
	updateEndpointCmd.Flags().BoolP("mutual-auth", "m", false, "Enable mutual authentication")
	updateEndpointCmd.Flags().BoolP("no-encryption", "e", false, "Disable encryption")
	updateEndpointCmd.Flags().StringP("kex-method", "k", "", "ASL key exchange method")
	updateEndpointCmd.Flags().StringP("cipher", "c", "", "Cipher configuration")
	endpointCli.AddCommand(updateEndpointCmd)

	deleteEndpointCmd.Flags().Int32P("id", "i", 0, "ID of the endpoint")
	deleteEndpointCmd.MarkFlagRequired("id")
	endpointCli.AddCommand(deleteEndpointCmd)

	listEndpointsCmd.Flags().StringP("version_number", "v", "", "Reference to the version")
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
		versionSetID, _ := cmd.Flags().GetString("version_number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.CreateEndpointConfigRequest{
			Name:                 name,
			MutualAuth:           mutualAuth,
			NoEncryption:         noEncryption,
			AslKeyExchangeMethod: v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[kexMethod]),
			Cipher:               &cipher,
			CreatedBy:            createdBy,
			VersionSetId:         versionSetID,
		}

		rsp, err := client.CreateEndpointConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create endpoint")
		}

		log.Info().Msgf("Endpoint created: %v", rsp)
		return nil
	},
}

var readEndpointCmd = &cobra.Command{
	Use:   "read",
	Short: "Read endpoint details",
	Long:  "Read and display details of a specific endpoint",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.GetEndpointConfigRequest{
			Id: id,
		}

		rsp, err := client.GetEndpointConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get endpoint")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp, "", outputFormat)
			return nil
		}

		PrintEndpointsAsTable([]*v1.EndpointConfig{rsp})
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
		mutualAuth, _ := cmd.Flags().GetBool("mutual-auth")
		noEncryption, _ := cmd.Flags().GetBool("no-encryption")
		kexMethod, _ := cmd.Flags().GetString("kex-method")
		cipher, _ := cmd.Flags().GetString("cipher")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		var kexMethodEnum v1.AslKeyexchangeMethod
		if kexMethod != "" {
			kexMethodEnum = v1.AslKeyexchangeMethod(v1.AslKeyexchangeMethod_value[kexMethod])
		}

		request := &v1.UpdateEndpointConfigRequest{
			Id:                   id,
			Name:                 &name,
			MutualAuth:           &mutualAuth,
			NoEncryption:         &noEncryption,
			AslKeyExchangeMethod: &kexMethodEnum,
			Cipher:               &cipher,
		}

		_, err = client.UpdateEndpointConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to update endpoint")
		}

		log.Info().Msgf("Endpoint updated successfully")
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
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.DeleteEndpointConfigRequest{
			Id: id,
		}

		_, err = client.DeleteEndpointConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to delete endpoint")
		}

		log.Info().Msgf("Endpoint deleted successfully")
		return nil
	},
}

var listEndpointsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all endpoints",
	Long:  "List all endpoints in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetID, _ := cmd.Flags().GetString("version_number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.ListEndpointConfigsRequest{
			VersionSetId: versionSetID,
		}

		rsp, err := client.ListEndpointConfigs(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list endpoints")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.Configs, "", outputFormat)
			return nil
		}

		PrintEndpointsAsTable(rsp.Configs)
		return nil
	},
}

func PrintEndpointsAsTable(endpoints []*v1.EndpointConfig) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "Version Set ID\t\t\tID\tNAME\tMUTUAL AUTH\tNO ENCRYPTION\tKEX METHOD\tCIPHER\tCREATED BY")

	for _, ep := range endpoints {
		fmt.Fprintf(w, "%s\t\t\t%d\t%s\t%t\t%t\t%s\t%s\t%s\n",
			ep.VersionSetId,
			ep.Id,
			ep.Name,
			ep.MutualAuth,
			ep.NoEncryption,
			ep.AslKeyExchangeMethod.String(),
			*ep.Cipher,
			ep.CreatedBy,
		)
	}
	w.Flush()
}
