package cli

import (
	"strings"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/spf13/cobra"
)

func init() {
	cli_logger.Debug().Msg("Registering cert_req commands")
	cert_req.Flags().StringP("node-serial", "n", "", "Node serial number")
	cert_req.MarkFlagRequired("node-serial")

	cert_req.Flags().StringP("plane", "p", "", "specify the plane of the certificate request either \"dataplane\" or \"controlplane\"")
	cert_req.MarkFlagRequired("plane")

	// Add new optional flags
	cert_req.Flags().String("algo", "", "Optional algorithm to use for certificate: rsa2048, rsa3072, rsa4096,secp256, secp384, secp521, ed25519, ed448, mldsa44, mldsa65, mldsa87, falcon512, falcon1024")
	cert_req.Flags().String("alt-algo", "", "Optional algorithm to use for certificate: rsa2048, rsa3072, rsa4096,secp256, secp384, secp521, ed25519, ed448, mldsa44, mldsa65, mldsa87, falcon512, falcon1024")

	rootCmd.AddCommand(cert_req)

	cert_fleet.Flags().StringP("version-set-id", "v", "", "Version set id")
	cert_fleet.MarkFlagRequired("version-set-id")

	cert_fleet.Flags().StringP("plane", "p", "", "specify the plane of the certificate request either \"dataplane\" or \"controlplane\"")
	cert_fleet.MarkFlagRequired("plane")

	// Add new optional flags
	cert_fleet.Flags().String("algo", "", "Optional algorithm to use for certificate: rsa2048, rsa3072, rsa4096,secp256, secp384, secp521, ed25519, ed448, mldsa44, mldsa65, mldsa87, falcon512, falcon1024")
	cert_fleet.Flags().String("alt-algo", "", "Optional algorithm to use for certificate: rsa2048, rsa3072, rsa4096,secp256, secp384, secp521, ed25519, ed448, mldsa44, mldsa65, mldsa87, falcon512, falcon1024")
	rootCmd.AddCommand(cert_fleet)

}

var cert_req = &cobra.Command{
	Use:   "cert_req",
	Short: "trigger a certificate request",
	Long:  "kritis3m_scale sends via control plane a certificate request to the node causing the node to send a csr to the est server",
	RunE: func(cmd *cobra.Command, args []string) error {

		nodeSerial, _ := cmd.Flags().GetString("node-serial")
		plane, _ := cmd.Flags().GetString("plane")
		if plane != "dataplane" && plane != "controlplane" {
			cli_logger.Fatal().Msg("Invalid plane specified. Please use either \"dataplane\" or \"controlplane\".")
		}

		// Get the new optional flags
		algo, _ := cmd.Flags().GetString("algo")
		altAlgo, _ := cmd.Flags().GetString("alt-algo")

		ctx, client, conn, cancel, err := getClient()
		defer conn.Close()
		defer cancel()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		southbound_plane := grpc_southbound.CertType_value[strings.ToUpper(plane)]

		//generate request
		req := &grpc_southbound.TriggerCertReqRequest{
			SerialNumber: nodeSerial,
			CertType:     grpc_southbound.CertType(southbound_plane),
		}

		// Add optional parameters if provided
		if algo != "" {
			req.Algo = &algo
		} else {
			req.Algo = nil
		}
		if altAlgo != "" {
			req.AltAlgo = &altAlgo
		} else {
			req.AltAlgo = nil
		}

		//send request
		resp, err := client.TriggerCertReq(ctx, req)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to send certificate request")
		}
		cli_logger.Info().Msgf("Returncode is %d", resp.Retcode)
		return nil
	},
}

var cert_fleet = &cobra.Command{
	Use:   "fleet_cert_req",
	Short: "trigger a certificate request for a fleet",
	Long:  "kritis3m_scale sends via control plane a certificate request to all nodes in the fleet causing the nodes to send a csr to the est server. Nodes not activated are not included",
	RunE: func(cmd *cobra.Command, args []string) error {

		versionSetId, _ := cmd.Flags().GetString("version-set-id")

		plane, _ := cmd.Flags().GetString("plane")
		if plane != "dataplane" && plane != "controlplane" {
			cli_logger.Fatal().Msg("Invalid plane specified. Please use either \"dataplane\" or \"controlplane\".")
		}

		// Get the new optional flags
		algo, _ := cmd.Flags().GetString("algo")
		altAlgo, _ := cmd.Flags().GetString("alt-algo")

		ctx, client, conn, cancel, err := getClient()
		defer conn.Close()
		defer cancel()
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to get client")
		}

		southbound_plane := grpc_southbound.CertType_value[strings.ToUpper(plane)]

		//generate request
		req := &grpc_southbound.TriggerFleetCertRequest{
			VersionSetId: versionSetId,
			CertType:     grpc_southbound.CertType(southbound_plane),
		}

		// Add optional parameters if provided
		if algo != "" {
			req.Algo = &algo
		} else {
			req.Algo = nil
		}
		if algo != "" && altAlgo != "" {
			req.AltAlgo = &altAlgo
		} else {
			req.AltAlgo = nil
		}

		//send request
		resp, err := client.TriggerFleetCertReq(ctx, req)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("Failed to send certificate request")
		}
		cli_logger.Info().Msgf("Returncode is %d", resp.Retcode)
		return nil
	},
}
