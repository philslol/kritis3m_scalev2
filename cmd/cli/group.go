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

	createGroupCmd.Flags().Int32P("log-level", "l", 0, "Log level for the group")
	createGroupCmd.MarkFlagRequired("log-level")

	createGroupCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	createGroupCmd.MarkFlagRequired("version-number")

	createGroupCmd.Flags().StringP("endpoint-config", "e", "", "Endpoint config name")
	createGroupCmd.MarkFlagRequired("endpoint-config")

	createGroupCmd.Flags().StringP("legacy-config", "c", "", "Legacy config name")
	createGroupCmd.Flags().StringP("created-by", "u", "", "User creating the group")
	createGroupCmd.MarkFlagRequired("created-by")
	groupCli.AddCommand(createGroupCmd)

	// Read command flags
	readGroupCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	readGroupCmd.MarkFlagRequired("version-number")
	readGroupCmd.Flags().StringP("name", "n", "", "Name of the group")
	readGroupCmd.MarkFlagRequired("name")
	readGroupCmd.Flags().BoolP("include", "i", false, "Include related endpoints")
	groupCli.AddCommand(readGroupCmd)

	// Update command flags
	updateGroupCmd.Flags().Int32P("id", "i", 0, "ID of the group")
	updateGroupCmd.MarkFlagRequired("id")
	updateGroupCmd.Flags().Int32P("log-level", "l", 0, "Log level for the group")
	updateGroupCmd.Flags().StringP("endpoint-config-name", "e", "", "Endpoint config name")
	updateGroupCmd.Flags().StringP("legacy-config-name", "c", "", "Legacy config name")
	updateGroupCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	groupCli.AddCommand(updateGroupCmd)

	// Delete command flags
	deleteGroupCmd.Flags().Int32P("id", "i", 0, "ID of the group")
	deleteGroupCmd.MarkFlagRequired("id")
	deleteGroupCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	deleteGroupCmd.MarkFlagRequired("version-number")
	groupCli.AddCommand(deleteGroupCmd)

	// List command flags
	listGroupsCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	listGroupsCmd.Flags().BoolP("include", "i", false, "Include related endpoints")
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
		logLevel, _ := cmd.Flags().GetInt32("log-level")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		endpointConfig, _ := cmd.Flags().GetString("endpoint-config")
		legacyConfig, _ := cmd.Flags().GetString("legacy-config")
		createdBy, _ := cmd.Flags().GetString("created-by")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		request := &v1.CreateGroupRequest{
			Name:               name,
			LogLevel:           logLevel,
			VersionSetId:       versionSetID,
			EndpointConfigName: endpointConfig,
			LegacyConfigName:   &legacyConfig,
			CreatedBy:          createdBy,
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
		versionSetID, _ := cmd.Flags().GetString("version-number")
		name, _ := cmd.Flags().GetString("name")

		id, _ := cmd.Flags().GetInt32("id")

		includeEndpoints, _ := cmd.Flags().GetBool("include")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		var request *v1.GetGroupRequest
		if id != 0 {
			request = &v1.GetGroupRequest{
				Query: &v1.GetGroupRequest_Id{
					Id: id,
				},
				IncludeEndpoints: includeEndpoints,
			}

		} else if versionSetID != "" && name != "" {
			request = &v1.GetGroupRequest{
				Query: &v1.GetGroupRequest_GroupQuery{
					GroupQuery: &v1.GroupNameQuery{
						VersionSetId: versionSetID,
						GroupName:    name,
					},
				},
				IncludeEndpoints: includeEndpoints,
			}
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
	},
}

var updateGroupCmd = &cobra.Command{
	Use:   "update",
	Short: "Update group details",
	Long:  "Update the details of an existing group",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		name, _ := cmd.Flags().GetString("name")

		logLevel, _ := cmd.Flags().GetInt32("log-level")
		endpointConfigName, _ := cmd.Flags().GetString("endpoint-config-name")
		legacyConfigName, _ := cmd.Flags().GetString("legacy-config-name")
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		var request *v1.UpdateGroupRequest
		if id != 0 {
			request = &v1.UpdateGroupRequest{
				Query: &v1.UpdateGroupRequest_Id{
					Id: id,
				},
			}
		} else if versionSetID != "" && name != "" {
			request = &v1.UpdateGroupRequest{
				Query: &v1.UpdateGroupRequest_GroupQuery{
					GroupQuery: &v1.GroupNameQuery{
						VersionSetId: versionSetID,
						GroupName:    name,
					},
				},
			}
		} else {
			log.Fatal().Msg("Must specify either id or version-number and name")
		}

		if cmd.Flags().Changed("endpoint-config-name") {
			request.EndpointConfigName = &endpointConfigName
		}

		if cmd.Flags().Changed("legacy-config-name") {
			request.LegacyConfigName = &legacyConfigName
		}

		if cmd.Flags().Changed("log-level") {
			request.LogLevel = &logLevel
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
		versionSetID, _ := cmd.Flags().GetString("version-number")
		includeEndpoints, _ := cmd.Flags().GetBool("include")
		var request *v1.ListGroupsRequest

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}
		defer cancel()
		defer conn.Close()

		if versionSetID != "" {
			request = &v1.ListGroupsRequest{
				VersionSetId:     &versionSetID,
				IncludeEndpoints: includeEndpoints,
			}
		} else {
			request = &v1.ListGroupsRequest{
				IncludeEndpoints: includeEndpoints,
			}
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

func PrintGroupResponseAsTable(groups []*v1.GroupResponse) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION SET ID\tLOG LEVEL\tENDPOINT CONFIG\tLEGACY CONFIG\tID")

	for _, groupResp := range groups {
		group := groupResp.GetGroup()
		legacyConfig := ""
		if group.LegacyConfigName != nil {
			legacyConfig = *group.LegacyConfigName
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\t%d\n",
			group.Name,
			group.VersionSetId,
			group.LogLevel,
			group.EndpointConfigName,
			legacyConfig,
			group.Id,
		)
	}
	w.Flush()
}
