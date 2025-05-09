package cli

import (
	"strings"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var cert_req = &cobra.Command{
	Use:   "cert_req",
	Short: "trigger a certificate request",
	Long:  "kritis3m_scale sends via control plane a certificate request to the node causing the node to send a csr to the est server",
	RunE: func(cmd *cobra.Command, args []string) error {

		nodeSerial, _ := cmd.Flags().GetString("node-serial")
		plane, _ := cmd.Flags().GetString("plane")
		if plane != "dataplane" && plane != "controlplane" {
			log.Fatal().Msg("Invalid plane specified. Please use either \"dataplane\" or \"controlplane\".")
		}

		ctx, client, conn, cancel, err := getClient()
		defer conn.Close()
		defer cancel()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to get client")
		}

		southbound_plane := grpc_southbound.CertType_value[strings.ToUpper(plane)]
		//generate request
		req := &grpc_southbound.TriggerCertReqRequest{
			SerialNumber: nodeSerial,
			CertType:     grpc_southbound.CertType(southbound_plane),
		}

		//send request
		resp, err := client.TriggerCertReq(ctx, req)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to send certificate request")
		}
		log.Info().Msgf("Returncode is %d", resp.Retcode)
		return nil
	},
}

func init() {
	log.Debug().Msg("Registering cert_req commands")
	cert_req.Flags().StringP("node-serial", "n", "", "Node serial number")
	cert_req.MarkFlagRequired("node-serial")

	cert_req.Flags().StringP("plane", "p", "", "specify the plane of the certificate request either \"dataplane\" or \"controlplane\"")
	cert_req.MarkFlagRequired("plane")
	rootCmd.AddCommand(cert_req)

	// Create command flags
}
