package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/spf13/cobra"
)

var proxyCli = &cobra.Command{
	Use:   "proxy",
	Short: "Manage proxies",
	Long:  "Create, read, update, and delete proxies in the system",
}

func init() {
	cli_logger.Debug().Msg("Registering proxy commands")
	rootCmd.AddCommand(proxyCli)

	// Create command flags
	createProxyCmd.Flags().StringP("node-serial", "n", "", "Serial number of the node")
	createProxyCmd.MarkFlagRequired("node-serial")

	createProxyCmd.Flags().StringP("group-name", "g", "", "Name of the group")
	createProxyCmd.MarkFlagRequired("group-name")

	createProxyCmd.Flags().BoolP("state", "s", true, "State of the proxy")

	createProxyCmd.Flags().StringP("proxy-type", "t", "FORWARD", "Type of proxy (FORWARD, REVERSE, TLSTLS)")
	createProxyCmd.MarkFlagRequired("proxy-type")

	createProxyCmd.Flags().StringP("server-endpoint", "e", "", "Server endpoint address")
	createProxyCmd.MarkFlagRequired("server-endpoint")

	createProxyCmd.Flags().StringP("client-endpoint", "c", "", "Client endpoint address")
	createProxyCmd.MarkFlagRequired("client-endpoint")

	createProxyCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	createProxyCmd.MarkFlagRequired("version-number")

	createProxyCmd.Flags().StringP("created-by", "u", "", "User creating the proxy")
	createProxyCmd.MarkFlagRequired("created-by")

	createProxyCmd.Flags().StringP("name", "m", "", "Name of the proxy")
	createProxyCmd.MarkFlagRequired("name")
	proxyCli.AddCommand(createProxyCmd)

	// Read command flags
	readProxyCmd.Flags().Int32P("id", "i", 0, "ID of the proxy")
	readProxyCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	readProxyCmd.Flags().StringP("name", "n", "", "Name of the proxy")
	readProxyCmd.Flags().StringP("serial", "s", "", "Serial number of the node")
	proxyCli.AddCommand(readProxyCmd)

	// Update command flags
	updateProxyCmd.Flags().Int32P("id", "i", 0, "ID of the proxy")
	updateProxyCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	updateProxyCmd.Flags().StringP("name", "n", "", "Name of the proxy")
	updateProxyCmd.Flags().BoolP("state", "s", true, "State of the proxy")
	updateProxyCmd.Flags().StringP("proxy-type", "t", "", "Type of proxy (FORWARD, REVERSE, TLSTLS)")
	updateProxyCmd.Flags().StringP("server-endpoint", "e", "", "Server endpoint address")
	updateProxyCmd.Flags().StringP("client-endpoint", "c", "", "Client endpoint address")
	proxyCli.AddCommand(updateProxyCmd)

	// Delete command flags
	deleteProxyCmd.Flags().Int32P("id", "i", 0, "ID of the proxy")
	deleteProxyCmd.MarkFlagRequired("id")
	proxyCli.AddCommand(deleteProxyCmd)

	// List command flags
	listProxiesCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	proxyCli.AddCommand(listProxiesCmd)

	// Add output format flags
	readProxyCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
	listProxiesCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "Output format: json, json-line, yaml")
}

var createProxyCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new proxy",
	Long:  "Create a new proxy with the specified parameters",
	RunE: func(cmd *cobra.Command, args []string) error {
		nodeSerial, _ := cmd.Flags().GetString("node-serial")
		groupName, _ := cmd.Flags().GetString("group-name")
		state, _ := cmd.Flags().GetBool("state")
		proxyType, _ := cmd.Flags().GetString("proxy-type")
		serverEndpoint, _ := cmd.Flags().GetString("server-endpoint")
		clientEndpoint, _ := cmd.Flags().GetString("client-endpoint")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		createdBy, _ := cmd.Flags().GetString("created-by")
		name, _ := cmd.Flags().GetString("name")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		// Convert proxy type string to enum value
		proxyTypeEnum := grpc_southbound.ProxyType_FORWARD // Default to FORWARD
		switch strings.ToUpper(proxyType) {
		case "FORWARD":
			proxyTypeEnum = grpc_southbound.ProxyType_FORWARD
		case "REVERSE":
			proxyTypeEnum = grpc_southbound.ProxyType_REVERSE
		case "TLSTLS":
			proxyTypeEnum = grpc_southbound.ProxyType_TLSTLS
		default:
			return fmt.Errorf("invalid proxy type: %s", proxyType)
		}

		request := &grpc_southbound.CreateProxyRequest{
			NodeSerialNumber:   nodeSerial,
			GroupName:          groupName,
			State:              state,
			ProxyType:          proxyTypeEnum,
			ServerEndpointAddr: serverEndpoint,
			ClientEndpointAddr: clientEndpoint,
			VersionSetId:       versionSetID,
			CreatedBy:          createdBy,
			Name:               name,
		}

		rsp, err := client.CreateProxy(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to create proxy")
		}

		cli_logger.Info().Msgf("Proxy created: %v", rsp)
		return nil
	},
}

var readProxyCmd = &cobra.Command{
	Use:   "read",
	Short: "Read proxy details",
	Long:  "Read and display details of a specific proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		name, _ := cmd.Flags().GetString("name")
		serial, _ := cmd.Flags().GetString("serial")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.GetProxyRequest{}

		if id != 0 {
			request.Query = &grpc_southbound.GetProxyRequest_Id{Id: id}
		} else if versionSetID != "" && name != "" {
			request.Query = &grpc_southbound.GetProxyRequest_NameQuery{
				NameQuery: &grpc_southbound.ProxyNameQuery{
					VersionSetId: versionSetID,
					Name:         name,
				},
			}
		} else if versionSetID != "" && serial != "" {
			request.Query = &grpc_southbound.GetProxyRequest_SerialQuery{
				SerialQuery: &grpc_southbound.ProxySerialQuery{
					VersionSetId: versionSetID,
					Serial:       serial,
				},
			}
		} else if versionSetID != "" {
			request.Query = &grpc_southbound.GetProxyRequest_VersionSetId{
				VersionSetId: versionSetID,
			}
		}
		rsp, err := client.GetProxy(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get proxy")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp, "", outputFormat)
			return nil
		}

		PrintProxiesAsTable(rsp.Proxy)
		return nil
	},
}

var updateProxyCmd = &cobra.Command{
	Use:   "update",
	Short: "Update proxy details",
	Long:  "Update the details of an existing proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		name, _ := cmd.Flags().GetString("name")
		state, _ := cmd.Flags().GetBool("state")
		proxyType, _ := cmd.Flags().GetString("proxy-type")
		serverEndpoint, _ := cmd.Flags().GetString("server-endpoint")
		clientEndpoint, _ := cmd.Flags().GetString("client-endpoint")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.UpdateProxyRequest{
			State:              &state,
			ServerEndpointAddr: &serverEndpoint,
			ClientEndpointAddr: &clientEndpoint,
		}

		if id != 0 {
			request.Query = &grpc_southbound.UpdateProxyRequest_Id{Id: id}
		} else if versionSetID != "" && name != "" {
			request.Query = &grpc_southbound.UpdateProxyRequest_NameQuery{
				NameQuery: &grpc_southbound.ProxyNameQuery{
					VersionSetId: versionSetID,
					Name:         name,
				},
			}
		} else {
			return fmt.Errorf("must specify either id or version-number and name")
		}

		if proxyType != "" {
			var pt grpc_southbound.ProxyType
			switch strings.ToUpper(proxyType) {
			case "FORWARD":
				pt = grpc_southbound.ProxyType_FORWARD
			case "REVERSE":
				pt = grpc_southbound.ProxyType_REVERSE
			case "TLSTLS":
				pt = grpc_southbound.ProxyType_TLSTLS
			default:
				return fmt.Errorf("invalid proxy type: %s", proxyType)
			}
			request.ProxyType = &pt
		}

		_, err = client.UpdateProxy(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to update proxy")
		}

		cli_logger.Info().Msgf("Proxy updated successfully")
		return nil
	},
}

var deleteProxyCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a proxy",
	Long:  "Delete an existing proxy from the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.DeleteProxyRequest{
			Id: id,
		}

		_, err = client.DeleteProxy(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to delete proxy")
		}

		cli_logger.Info().Msgf("Proxy deleted successfully")
		return nil
	},
}

var listProxiesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all proxies",
	Long:  "List all proxies in the system",
	RunE: func(cmd *cobra.Command, args []string) error {
		versionSetID, _ := cmd.Flags().GetString("version-number")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &grpc_southbound.GetProxyRequest{}
		if versionSetID != "" {
			request.Query = &grpc_southbound.GetProxyRequest_NameQuery{
				NameQuery: &grpc_southbound.ProxyNameQuery{
					VersionSetId: versionSetID,
				},
			}
		}
		rsp, err := client.GetProxy(ctx, request)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to list proxies")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.Proxy, "", outputFormat)
			return nil
		}

		PrintProxiesAsTable(rsp.Proxy)
		return nil
	},
}

func PrintProxiesAsTable(proxies []*grpc_southbound.Proxy) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNODE SERIAL\tGROUP NAME\tSTATE\tPROXY TYPE\tSERVER ENDPOINT\tCLIENT ENDPOINT\tVERSION SET ID\tCREATED BY\tNAME")

	for _, proxy := range proxies {
		fmt.Fprintf(w, "%d\t%s\t%s\t%t\t%s\t%s\t%s\t%s\t%s\t%s\n",
			proxy.Id,
			proxy.NodeSerialNumber,
			proxy.GroupName,
			proxy.State,
			proxy.ProxyType.String(),
			proxy.ServerEndpointAddr,
			proxy.ClientEndpointAddr,
			proxy.VersionSetId,
			proxy.CreatedBy,
			proxy.Name,
		)
	}
	w.Flush()
}
