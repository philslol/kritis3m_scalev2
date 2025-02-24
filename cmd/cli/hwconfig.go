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
	log.Debug().Msg("Registering hardware config commands")
	rootCmd.AddCommand(hwConfigCli)

	// Create command flags
	createHwConfigCmd.Flags().StringP("device", "d", "", "Device name")
	createHwConfigCmd.MarkFlagRequired("device")

	createHwConfigCmd.Flags().StringP("ip-cidr", "i", "", "IP CIDR")
	createHwConfigCmd.MarkFlagRequired("ip-cidr")

	createHwConfigCmd.Flags().Int32P("node-id", "n", 0, "Node ID")
	createHwConfigCmd.MarkFlagRequired("node-id")

	createHwConfigCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	createHwConfigCmd.MarkFlagRequired("version-number")

	// Add all commands to hardware config CLI
	hwConfigCli.AddCommand(createHwConfigCmd)

	// Read command flags
	readHwConfigCmd.Flags().Int32P("id", "i", 0, "ID of the hardware config")
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
		nodeID, _ := cmd.Flags().GetInt32("node-id")
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.CreateHardwareConfigRequest{
			Device:       device,
			IpCidr:       ipCidr,
			NodeId:       nodeID,
			VersionSetId: versionSetID,
		}

		rsp, err := client.CreateHardwareConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create hardware config")
		}

		log.Info().Msgf("Hardware config created: %v", rsp)
		return nil
	},
}

var readHwConfigCmd = &cobra.Command{
	Use:   "read",
	Short: "Read hardware config details",
	Long:  "Read and display details of a specific hardware configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.GetHardwareConfigRequest{
			Id: id,
		}

		rsp, err := client.GetHardwareConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get hardware config")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.HardwareConfig, "", outputFormat)
			return nil
		}

		PrintHardwareConfigAsTable([]*v1.HardwareConfig{rsp.HardwareConfig})
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
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.UpdateHardwareConfigRequest{
			Id:           id,
			Device:       &device,
			IpCidr:       &ipCidr,
			VersionSetId: &versionSetID,
		}

		_, err = client.UpdateHardwareConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to update hardware config")
		}

		log.Info().Msg("Hardware config updated successfully")
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
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.DeleteHardwareConfigRequest{
			Id: id,
		}

		_, err = client.DeleteHardwareConfig(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to delete hardware config")
		}

		log.Info().Msg("Hardware config deleted successfully")
		return nil
	},
}

var listHwConfigsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all hardware configs",
	Long:  "List all hardware configurations in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.ListHardwareConfigsRequest{}

		rsp, err := client.ListHardwareConfigs(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list hardware configs")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.HardwareConfigs, "", outputFormat)
			return nil
		}

		PrintHardwareConfigAsTable(rsp.HardwareConfigs)
		return nil
	},
}

func PrintHardwareConfigAsTable(configs []*v1.HardwareConfig) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tDEVICE\tIP CIDR\tNODE ID\tVERSION SET ID")

	for _, config := range configs {
		fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s\n",
			config.Id,
			config.Device,
			config.IpCidr,
			config.NodeId,
			config.VersionSetId,
		)
	}
	w.Flush()
}
