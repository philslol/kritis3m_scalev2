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
	log.Debug().Msg("Registering group commands")
	rootCmd.AddCommand(groupCli)

	// Create command flags
	createGroupCmd.Flags().StringP("name", "n", "", "Name of the group")
	createGroupCmd.MarkFlagRequired("name")

	createGroupCmd.Flags().Int32P("endpoint-config-id", "e", 0, "Endpoint config ID")
	createGroupCmd.MarkFlagRequired("endpoint-config-id")

	createGroupCmd.Flags().StringP("version_number", "v", "", "Version set ID")
	createGroupCmd.MarkFlagRequired("version_number")

	createGroupCmd.Flags().Int32P("log-level", "l", 0, "Log level for the group")
	createGroupCmd.Flags().Int32P("legacy-config-id", "c", 0, "Legacy config ID")
	createGroupCmd.Flags().Int32P("id", "i", 0, "ID of the group")
	groupCli.AddCommand(createGroupCmd)

	// Read command flags
	readGroupCmd.Flags().Int32P("id", "i", 0, "ID of the group")
	readGroupCmd.Flags().StringP("version_number", "v", "", "Version set ID to list groups for")
	groupCli.AddCommand(readGroupCmd)

	// Update command flags
	updateGroupCmd.Flags().Int32P("id", "i", 0, "ID of the group")
	updateGroupCmd.MarkFlagRequired("id")
	updateGroupCmd.Flags().StringP("name", "n", "", "Name of the group")
	updateGroupCmd.Flags().StringP("version_number", "v", "", "Version set ID to list groups for")
	updateGroupCmd.Flags().Int32P("endpoint-config-id", "e", 0, "Endpoint config ID")
	updateGroupCmd.Flags().Int32P("log-level", "l", 0, "Log level for the group")
	updateGroupCmd.Flags().Int32P("legacy-config-id", "c", 0, "Legacy config ID")
	groupCli.AddCommand(updateGroupCmd)

	// Delete command flags
	deleteGroupCmd.Flags().Int32P("id", "i", 0, "ID of the group")
	deleteGroupCmd.MarkFlagRequired("id")
	groupCli.AddCommand(deleteGroupCmd)

	// List command flags
	groupCli.AddCommand(listGroupsCmd)
}

var groupCli = &cobra.Command{
	Use:   "group",
	Short: "Manage groups",
	Long:  "Create, read, update, and delete groups in the system",
}

var createGroupCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new group",
	Long:  "Create a new group with the specified parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		endpointConfigID, _ := cmd.Flags().GetInt32("endpoint-config-id")
		logLevel, _ := cmd.Flags().GetInt32("log-level")

		legacyConfigID, _ := cmd.Flags().GetInt32("legacy-config-id")
		var legacy *int32
		if legacyConfigID == 0 {
			legacy = nil
		} else {
			legacy = &legacyConfigID
		}

		versionSetID, _ := cmd.Flags().GetString("version_number")
		if versionSetID == "" {
			log.Error().Msg("no version set id provided")
			return fmt.Errorf("no version set id provided")
		}

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		request := &v1.CreateGroupRequest{
			Name:             name,
			EndpointConfigId: endpointConfigID,
			LegacyConfigId:   legacy,
			LogLevel:         logLevel,
			VersionSetId:     versionSetID,
		}

		if cmd.Flags().Changed("log-level") {
			request.LogLevel = logLevel
		}

		if cmd.Flags().Changed("legacy-config-id") {
			request.LegacyConfigId = &legacyConfigID
		}

		rsp, err := client.CreateGroup(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create group")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp, "", outputFormat)
			return nil
		}

		PrintGroupResponseAsTable([]*v1.GroupResponse{rsp})
		return nil
	},
}

var readGroupCmd = &cobra.Command{
	Use:   "read",
	Short: "Read group details",
	Long:  "Read and display details of a specific group",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		versionSetId, _ := cmd.Flags().GetString("version_number")

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
			request := &v1.GetGroupRequest{
				Id: id,
			}

			rsp, err := client.GetGroup(ctx, request)
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get group")
			}

			if HasMachineOutputFlag() {
				SuccessOutput(rsp, "", outputFormat)
				return nil
			}

			PrintGroupResponseAsTable([]*v1.GroupResponse{rsp})
			return nil
		}

		request := &v1.ListGroupsRequest{
			VersionSetId: &versionSetId,
		}

		rsp, err := client.ListGroups(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list groups")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.GetGroups(), "", outputFormat)
			return nil
		}

		PrintGroupResponseAsTable(rsp.GetGroups())
		return nil
	},
}

var updateGroupCmd = &cobra.Command{
	Use:   "update",
	Short: "Update group details",
	Long:  "Update the details of an existing group",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		name, _ := cmd.Flags().GetString("name")
		endpointConfigID, _ := cmd.Flags().GetInt32("endpoint-config-id")
		logLevel, _ := cmd.Flags().GetInt32("log-level")
		legacyConfigID, _ := cmd.Flags().GetInt32("legacy-config-id")
		versionSetID, _ := cmd.Flags().GetString("version_number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		request := &v1.UpdateGroupRequest{
			Id: id,
		}

		if cmd.Flags().Changed("name") {
			request.Name = &name
		}

		if cmd.Flags().Changed("endpoint-config-id") {
			request.EndpointConfigId = &endpointConfigID
		}

		if cmd.Flags().Changed("log-level") {
			request.LogLevel = &logLevel
		}
		if cmd.Flags().Changed("version_number") {
			request.VersionSetId = &versionSetID
		}

		if cmd.Flags().Changed("legacy-config-id") {
			request.LegacyConfigId = &legacyConfigID
		}

		_, err = client.UpdateGroup(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to update group")
		}

		log.Info().Msg("Group updated successfully")
		return nil
	},
}

var deleteGroupCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a group",
	Long:  "Delete an existing group from the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		request := &v1.DeleteGroupRequest{
			Id: id,
		}

		_, err = client.DeleteGroup(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to delete group")
		}

		log.Info().Msg("Group deleted successfully")
		return nil
	},
}

var listGroupsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all groups",
	Long:  "List all groups in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		request := &v1.ListGroupsRequest{}

		rsp, err := client.ListGroups(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list groups")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.GetGroups(), "", outputFormat)
			return nil
		}

		PrintGroupResponseAsTable(rsp.GetGroups())
		return nil
	},
}

func PrintGroupResponseAsTable(groups []*v1.GroupResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tLOG LEVEL\tENDPOINT CONFIG ID\tLEGACY CONFIG ID")

	for _, groupResp := range groups {
		group := groupResp.GetGroup()
		legacyConfigID := ""
		if group.LegacyConfigId != nil {
			legacyConfigID = fmt.Sprintf("%d", *group.LegacyConfigId)
		}
		fmt.Fprintf(w, "%d\t%s\t%d\t%d\t%s\n",
			group.Id,
			group.Name,
			group.LogLevel,
			group.EndpointConfigId,
			legacyConfigID,
		)
	}
	w.Flush()
}
