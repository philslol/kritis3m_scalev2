package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/spf13/cobra"
)

func init() {
	cli_logger.Debug().Msg("Registering version set commands")
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

	listVersionSetsCmd.Flags().StringP("state", "s", "", "State of the version set: DRAFT, PENDING_DEPLOYMENT, ACTIVE, DISABLED")
	versionSetCli.AddCommand(listVersionSetsCmd)

	// Add output format flags
	readVersionSetCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	listVersionSetsCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
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
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.CreateVersionSetRequest{
			Name:        name,
			Description: description,
			CreatedBy:   createdBy,
		}

		rsp, err := client.CreateVersionSet(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to create version set")
		}

		cli_logger.Info().Msgf("Version set created: %v", rsp)
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
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.GetVersionSetRequest{
			Id: id,
		}

		rsp, err := client.GetVersionSet(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to execute grpc get version set")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.GetVersionSet(), "", outputFormat)
			return nil
		}

		PrintVersionSetsAsTable([]*grpc_southbound.VersionSet{rsp.GetVersionSet()})
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
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.UpdateVersionSetRequest{
			Id:          id,
			Name:        name,
			Description: description,
		}

		rsp, err := client.UpdateVersionSet(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to update version set")
		}

		cli_logger.Info().Msgf("Version set updated: %v", rsp)
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
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.DeleteVersionSetRequest{
			Id: id,
		}

		rsp, err := client.DeleteVersionSet(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to delete version set")
		}

		cli_logger.Info().Msgf("Version set deleted: %v", rsp)
		return nil
	},
}

var listVersionSetsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all version sets",
	Long:  "List all version sets in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		state, _ := cmd.Flags().GetString("state")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.ListVersionSetsRequest{}

		if state != "" {
			vs := grpc_southbound.VersionState(grpc_southbound.VersionState_value[state])
			request.State = &vs
		}

		rsp, err := client.ListVersionSets(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to list version sets")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.GetVersionSets(), "", outputFormat)
			return nil
		}
		PrintVersionSetsAsTable(rsp.GetVersionSets())
		return nil
	},
}

func PrintVersionSetsAsTable(versionSets []*grpc_southbound.VersionSet) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tDESCRIPTION\tSTATE\tACTIVATED AT\tDISABLED AT\tCREATED BY")

	for _, vs := range versionSets {
		activatedAt := ""
		if vs.ActivatedAt != nil {
			activatedAt = vs.ActivatedAt.AsTime().Format(HeadscaleDateTimeFormat)
		}
		disabledAt := ""
		if vs.DisabledAt != nil {
			disabledAt = vs.DisabledAt.AsTime().Format(HeadscaleDateTimeFormat)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			vs.Id,
			vs.Name,
			vs.Description,
			vs.State.String(),
			activatedAt,
			disabledAt,
			vs.CreatedBy,
		)
	}
	w.Flush()
}
