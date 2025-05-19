package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/spf13/cobra"
)

func init() {
	cli_logger.Debug().Msg("Registering hardware config commands")
	rootCmd.AddCommand(hwConfigCli)

	// Create command flags
	createHwConfigCmd.Flags().StringP("device", "d", "", "Device name")
	createHwConfigCmd.MarkFlagRequired("device")

	createHwConfigCmd.Flags().StringP("ip-cidr", "i", "", "IP CIDR")
	createHwConfigCmd.MarkFlagRequired("ip-cidr")

	createHwConfigCmd.Flags().StringP("serial-number", "s", "", "Node serial number")
	createHwConfigCmd.MarkFlagRequired("serial-number")

	createHwConfigCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	createHwConfigCmd.MarkFlagRequired("version-number")

	createHwConfigCmd.Flags().StringP("created-by", "u", "", "User creating the hardware config")
	createHwConfigCmd.MarkFlagRequired("created-by")

	// Add all commands to hardware config CLI
	hwConfigCli.AddCommand(createHwConfigCmd)

	// Read command flags
	readHwConfigCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	readHwConfigCmd.MarkFlagRequired("version-number")
	readHwConfigCmd.Flags().StringP("serial-number", "s", "", "Node serial number")
	readHwConfigCmd.MarkFlagRequired("serial-number")
	readHwConfigCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	hwConfigCli.AddCommand(readHwConfigCmd)

	// Update command flags
	updateHwConfigCmd.Flags().Int32P("id", "i", 0, "ID of the hardware config")
	updateHwConfigCmd.MarkFlagRequired("id")
	updateHwConfigCmd.Flags().StringP("device", "d", "", "Device name")
	updateHwConfigCmd.Flags().StringP("ip-cidr", "c", "", "IP CIDR")
	updateHwConfigCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	hwConfigCli.AddCommand(updateHwConfigCmd)

	// Delete command flags
	deleteHwConfigCmd.Flags().Int32P("id", "i", 0, "ID of the hardware config")
	deleteHwConfigCmd.MarkFlagRequired("id")
	hwConfigCli.AddCommand(deleteHwConfigCmd)

	// List command flags
	listHwConfigsCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	listHwConfigsCmd.MarkFlagRequired("version-number")
	listHwConfigsCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	hwConfigCli.AddCommand(listHwConfigsCmd)
}

var hwConfigCli = &cobra.Command{
	Use:   "hwconfig",
	Short: "Manage hardware configurations",
	Long:  "Create, read, update, and delete hardware configurations in the system",
}

var createHwConfigCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new hardware configuration",
	Long:  "Create a new hardware configuration with the specified parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		device, _ := cmd.Flags().GetString("device")
		ipCidr, _ := cmd.Flags().GetString("ip-cidr")
		cli_logger.Info().Msgf("ipCidr: %s", ipCidr)
		nodeSerial, _ := cmd.Flags().GetString("serial-number")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		createdBy, _ := cmd.Flags().GetString("created-by")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.CreateHardwareConfigRequest{
			Device:           device,
			IpCidr:           ipCidr,
			NodeSerialNumber: nodeSerial,
			VersionSetId:     versionSetID,
			CreatedBy:        createdBy,
		}

		rsp, err := client.CreateHardwareConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to create hardware config")
		}

		cli_logger.Info().Msgf("Hardware config created: %v", rsp)
		return nil
	},
}

var readHwConfigCmd = &cobra.Command{
	Use:   "read",
	Short: "Read hardware config details",
	Long:  "Read and display details of a specific hardware configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetID, _ := cmd.Flags().GetString("version-number")
		nodeSerial, _ := cmd.Flags().GetString("serial-number")

		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		var request *grpc_southbound.GetHardwareConfigRequest
		if id != 0 {
			//create GetHardwareConfigRequest_Id
			request = &grpc_southbound.GetHardwareConfigRequest{
				Query: &grpc_southbound.GetHardwareConfigRequest_Id{
					Id: id,
				},
			}
		} else if versionSetID != "" && nodeSerial != "" {
			//create GetHardwareConfigRequest_HardwareConfigQuery
			request = &grpc_southbound.GetHardwareConfigRequest{
				Query: &grpc_southbound.GetHardwareConfigRequest_HardwareConfigQuery{
					HardwareConfigQuery: &grpc_southbound.HardwareConfigNameQuery{
						VersionSetId:     versionSetID,
						NodeSerialNumber: nodeSerial,
					},
				},
			}
		} else if versionSetID != "" && nodeSerial == "" {
			//create GetHardwareConfigRequest_VersionSetId
			request = &grpc_southbound.GetHardwareConfigRequest{
				Query: &grpc_southbound.GetHardwareConfigRequest_VersionSetId{
					VersionSetId: versionSetID,
				},
			}
		}

		rsp, err := client.GetHardwareConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get hardware config")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.HardwareConfig, "", outputFormat)
			return nil
		}

		PrintHardwareConfigAsTable(rsp.HardwareConfig)
		return nil
	},
}

var updateHwConfigCmd = &cobra.Command{
	Use:   "update",
	Short: "Update hardware config details",
	Long:  "Update the details of an existing hardware configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		device, _ := cmd.Flags().GetString("device")
		ipCidr, _ := cmd.Flags().GetString("ip-cidr")
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.UpdateHardwareConfigRequest{
			Id:           id,
			Device:       &device,
			IpCidr:       &ipCidr,
			VersionSetId: &versionSetID,
		}

		_, err = client.UpdateHardwareConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to update hardware config")
		}

		cli_logger.Info().Msg("Hardware config updated successfully")
		return nil
	},
}

var deleteHwConfigCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a hardware config",
	Long:  "Delete an existing hardware configuration from the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.DeleteHardwareConfigRequest{
			Id: id,
		}

		_, err = client.DeleteHardwareConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to delete hardware config")
		}

		cli_logger.Info().Msg("Hardware config deleted successfully")
		return nil
	},
}

var listHwConfigsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all hardware configs",
	Long:  "List all hardware configurations in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.GetHardwareConfigRequest{
			Query: &grpc_southbound.GetHardwareConfigRequest_VersionSetId{
				VersionSetId: versionSetID,
			},
		}

		rsp, err := client.GetHardwareConfig(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to list hardware configs")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.HardwareConfig, "", outputFormat)
			return nil
		}

		PrintHardwareConfigAsTable(rsp.HardwareConfig)
		return nil
	},
}

func PrintHardwareConfigAsTable(configs []*grpc_southbound.HardwareConfig) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "DEVICE\tIP CIDR\tNODE SERIAL\tVERSION SET ID\tCREATED BY")

	for _, config := range configs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			config.Device,
			config.IpCidr,
			config.NodeSerialNumber,
			config.VersionSetId,
			config.CreatedBy,
		)
	}
	w.Flush()
}
