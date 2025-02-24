package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	v1 "github.com/philslol/kritis3m_scalev2/gen/go/v1"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var proxyCli = &cobra.Command{
	Use:   "proxy",
	Short: "Manage proxies",
	Long:  "Create, read, update, and delete proxies in the system",
}

func init() {
	log.Debug().Msg("Registering proxy commands")
	rootCmd.AddCommand(proxyCli)

	// Create command flags
	createProxyCmd.Flags().Int32P("node-id", "n", 0, "ID of the node")
	createProxyCmd.MarkFlagRequired("node-id")
	createProxyCmd.Flags().Int32P("group-id", "g", 0, "ID of the group")
	createProxyCmd.MarkFlagRequired("group-id")
	createProxyCmd.Flags().BoolP("state", "s", true, "State of the proxy")

	createProxyCmd.Flags().StringP("proxy-type", "t", "FORWARD", "Type of proxy (FORWARD, REVERSE, TLSTLS)")
	createProxyCmd.MarkFlagRequired("proxy-type")

	createProxyCmd.Flags().StringP("server-endpoint", "e", "", "Server endpoint address")
	createProxyCmd.MarkFlagRequired("server-endpoint")

	createProxyCmd.Flags().StringP("client-endpoint", "c", "", "Client endpoint address")
	createProxyCmd.MarkFlagRequired("client-endpoint")
	createProxyCmd.Flags().StringP("version-number", "v", "", "Version set ID")
	createProxyCmd.Flags().StringP("created-by", "u", "", "User creating the proxy")
	proxyCli.AddCommand(createProxyCmd)

	readProxyCmd.Flags().Int32P("id", "i", 0, "ID of the proxy")
	readProxyCmd.MarkFlagRequired("id")
	proxyCli.AddCommand(readProxyCmd)

	updateProxyCmd.Flags().Int32P("id", "i", 0, "ID of the proxy")
	updateProxyCmd.MarkFlagRequired("id")
	updateProxyCmd.Flags().BoolP("state", "s", true, "State of the proxy")
	updateProxyCmd.Flags().StringP("proxy-type", "t", "", "Type of proxy (FORWARD, REVERSE, TLSTLS)")
	updateProxyCmd.Flags().StringP("server-endpoint", "e", "", "Server endpoint address")
	updateProxyCmd.Flags().StringP("client-endpoint", "c", "", "Client endpoint address")
	proxyCli.AddCommand(updateProxyCmd)

	deleteProxyCmd.Flags().Int32P("id", "i", 0, "ID of the proxy")
	deleteProxyCmd.MarkFlagRequired("id")
	proxyCli.AddCommand(deleteProxyCmd)

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
		nodeID, _ := cmd.Flags().GetInt32("node-id")
		groupID, _ := cmd.Flags().GetInt32("group-id")
		state, _ := cmd.Flags().GetBool("state")
		proxyType, _ := cmd.Flags().GetString("proxy-type")
		serverEndpoint, _ := cmd.Flags().GetString("server-endpoint")
		clientEndpoint, _ := cmd.Flags().GetString("client-endpoint")
		versionSetID, _ := cmd.Flags().GetString("version-number")
		createdBy, _ := cmd.Flags().GetString("created-by")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		// Convert proxy type string to enum value
		proxyTypeEnum := v1.ProxyType_FORWARD // Default to FORWARD
		switch strings.ToUpper(proxyType) {
		case "FORWARD":
			proxyTypeEnum = v1.ProxyType_FORWARD
		case "REVERSE":
			proxyTypeEnum = v1.ProxyType_REVERSE
		case "TLSTLS":
			proxyTypeEnum = v1.ProxyType_TLSTLS
		default:
			return fmt.Errorf("invalid proxy type: %s", proxyType)
		}

		request := &v1.CreateProxyRequest{
			NodeId:             nodeID,
			GroupId:            groupID,
			State:              state,
			ProxyType:          proxyTypeEnum,
			ServerEndpointAddr: serverEndpoint,
			ClientEndpointAddr: clientEndpoint,
			VersionSetId:       versionSetID,
			CreatedBy:          createdBy,
		}

		rsp, err := client.CreateProxy(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to create proxy")
		}

		log.Info().Msgf("Proxy created: %v", rsp)
		return nil
	},
}

var readProxyCmd = &cobra.Command{
	Use:   "read",
	Short: "Read proxy details",
	Long:  "Read and display details of a specific proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.GetProxyRequest{
			Id: id,
		}

		rsp, err := client.GetProxy(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get proxy")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp, "", outputFormat)
			return nil
		}

		PrintProxiesAsTable([]*v1.Proxy{rsp.Proxy})
		return nil
	},
}

var updateProxyCmd = &cobra.Command{
	Use:   "update",
	Short: "Update proxy details",
	Long:  "Update the details of an existing proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, _ := cmd.Flags().GetInt32("id")
		state, _ := cmd.Flags().GetBool("state")
		proxyType, _ := cmd.Flags().GetString("proxy-type")
		serverEndpoint, _ := cmd.Flags().GetString("server-endpoint")
		clientEndpoint, _ := cmd.Flags().GetString("client-endpoint")

		ctx, client, conn, cancel, err := getClient()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		var proxyTypePtr *v1.ProxyType
		if proxyType != "" {
			// Convert proxy type string to enum value
			var pt v1.ProxyType
			switch strings.ToUpper(proxyType) {
			case "FORWARD":
				pt = v1.ProxyType_FORWARD
			case "REVERSE":
				pt = v1.ProxyType_REVERSE
			case "TLSTLS":
				pt = v1.ProxyType_TLSTLS
			default:
				return fmt.Errorf("invalid proxy type: %s", proxyType)
			}
			proxyTypePtr = &pt
		}

		request := &v1.UpdateProxyRequest{
			Id:                 id,
			State:              &state,
			ProxyType:          proxyTypePtr,
			ServerEndpointAddr: &serverEndpoint,
			ClientEndpointAddr: &clientEndpoint,
		}

		_, err = client.UpdateProxy(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to update proxy")
		}

		log.Info().Msgf("Proxy updated successfully")
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
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.DeleteProxyRequest{
			Id: id,
		}

		_, err = client.DeleteProxy(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to delete proxy")
		}

		log.Info().Msgf("Proxy deleted successfully")
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
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		defer cancel()
		defer conn.Close()

		request := &v1.ListProxiesRequest{
			VersionSetId: versionSetID,
		}

		rsp, err := client.ListProxies(ctx, request)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to list proxies")
		}

		if HasMachineOutputFlag() {
			SuccessOutput(rsp.Proxies, "", outputFormat)
			return nil
		}

		PrintProxiesAsTable(rsp.Proxies)
		return nil
	},
}

func PrintProxiesAsTable(proxies []*v1.Proxy) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tNODE ID\tGROUP ID\tSTATE\tPROXY TYPE\tSERVER ENDPOINT\tCLIENT ENDPOINT\tVERSION SET ID\tCREATED BY")

	for _, proxy := range proxies {
		fmt.Fprintf(w, "%d\t%d\t%d\t%t\t%s\t%s\t%s\t%s\t%s\n",
			proxy.Id,
			proxy.NodeId,
			proxy.GroupId,
			proxy.State,
			proxy.ProxyType.String(),
			proxy.ServerEndpointAddr,
			proxy.ClientEndpointAddr,
			proxy.VersionSetId,
			proxy.CreatedBy,
		)
	}
	w.Flush()
}
