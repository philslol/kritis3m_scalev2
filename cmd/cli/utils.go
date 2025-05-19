package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"

	grpc_southbound "github.com/Laboratory-for-Safe-and-Secure-Systems/kritis3m_proto/southbound"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/philslol/kritis3m_scalev2/control"
	"github.com/philslol/kritis3m_scalev2/control/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

const (
	HeadscaleDateTimeFormat = "2006-01-02 15:04:05"
	SocketWritePermissions  = 0o666
)

func getClient() (context.Context, grpc_southbound.SouthboundClient, *grpc.ClientConn, context.CancelFunc, error) {
	cfg, err := types.GetKritis3mScaleConfig()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to load configuration while creating headscale instance: %w", err)
	}

	cli_logger.Debug().
		Dur("timeout", cfg.CliConfig.Timeout).
		Msgf("Setting timeout")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.CliConfig.Timeout)

	grpcOptions := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	address := cfg.CliConfig.ServerAddr
	cli_logger.Trace().Caller().Str("address", address).Msg("Connecting via gRPC")

	conn, err := grpc.NewClient(address, grpcOptions...)
	if err != nil {
		cli_logger.Fatal().Caller().Err(err).Msgf("Could not connect: %v", err)
		os.Exit(-1) // we get here if logging is suppressed (i.e., json output)
	}

	client := grpc_southbound.NewSouthboundClient(conn)

	return ctx, client, conn, cancel, nil
}

func getKritis3mScaleApp() (*control.Kritis3m_Scale, error) {
	cfg, err := types.GetKritis3mScaleConfig()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to load configuration while creating headscale instance: %w",
			err,
		)
	}

	app, err := control.NewKritis3m_scale(cfg)
	if err != nil {
		return nil, err
	}

	return app, nil
}

func ErrorOutput(errResult error, override string, outputFormat string) {
	type errOutput struct {
		Error string `json:"error"`
	}

	SuccessOutput(errOutput{errResult.Error()}, override, outputFormat)
}

func HasMachineOutputFlag() bool {
	for _, arg := range os.Args {
		if arg == "json" || arg == "json-line" || arg == "yaml" {
			return true
		}
	}

	return false
}

func SuccessOutput(result interface{}, override string, outputFormat string) {
	var jsonBytes []byte
	var err error
	switch outputFormat {
	case "json":
		jsonBytes, err = json.MarshalIndent(result, "", "\t")
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("failed to unmarshal output")
		}
	case "json-line":
		jsonBytes, err = json.Marshal(result)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("failed to unmarshal output")
		}
	case "yaml":
		jsonBytes, err = yaml.Marshal(result)
		if err != nil {
			cli_logger.Fatal().Err(err).Msg("failed to unmarshal output")
		}
	default:
		//nolint
		fmt.Println(override)

		return
	}

	//nolint
	fmt.Println(string(jsonBytes))
}

type TableColumn struct {
	Header    string
	FieldPath string
}

func PrintAsTable(items interface{}, columns []TableColumn) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

	// Print headers
	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	// Get the slice value using reflection
	val := reflect.ValueOf(items)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Slice {
		cli_logger.Error().Msg("Input must be a slice")
		return
	}

	// Iterate over each item in the slice
	for i := 0; i < val.Len(); i++ {
		item := val.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}

		// Build row values
		values := make([]string, len(columns))
		for j, col := range columns {
			fieldValue := getFieldByPath(item, strings.Split(col.FieldPath, "."))
			values[j] = formatValue(fieldValue)
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	w.Flush()
}

func getFieldByPath(v reflect.Value, path []string) reflect.Value {
	for _, fieldName := range path {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() != reflect.Struct {
			return reflect.Value{}
		}
		v = v.FieldByName(fieldName)
		if !v.IsValid() {
			return reflect.Value{}
		}
	}
	return v
}

func formatValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return ""
		}
		return formatValue(v.Elem())
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Bool:
		return fmt.Sprintf("%v", v.Bool())
	case reflect.Struct:
		// Handle special types
		if v.Type().String() == "*timestamp.Timestamp" {
			if ts, ok := v.Interface().(*timestamp.Timestamp); ok && ts != nil {
				return ts.AsTime().Format(HeadscaleDateTimeFormat)
			}
			return ""
		}
		// Handle protobuf enums
		if v.MethodByName("String").IsValid() {
			return v.MethodByName("String").Call(nil)[0].String()
		}
		return fmt.Sprintf("%v", v.Interface())
	default:
		return fmt.Sprintf("%v", v.Interface())
	}
}

// Example usage for Node type
func PrintNodesAsTable(nodes []*grpc_southbound.Node) {
	columns := []TableColumn{
		{Header: "ID", FieldPath: "Id"},
		{Header: "SERIAL NUMBER", FieldPath: "SerialNumber"},
		{Header: "NETWORK INDEX", FieldPath: "NetworkIndex"},
		{Header: "LOCALITY", FieldPath: "Locality"},
		{Header: "LAST SEEN", FieldPath: "LastSeen"},
		{Header: "VERSION SET ID", FieldPath: "VersionSetId"},
	}
	PrintAsTable(nodes, columns)
}

// Example usage for VersionSet type using the generic function
func PrintVersionSetsAsTableGeneric(versionSets []*grpc_southbound.VersionSet) {
	columns := []TableColumn{
		{Header: "ID", FieldPath: "Id"},
		{Header: "NAME", FieldPath: "Name"},
		{Header: "DESCRIPTION", FieldPath: "Description"},
		{Header: "STATE", FieldPath: "State"},
		{Header: "ACTIVATED AT", FieldPath: "ActivatedAt"},
		{Header: "DISABLED AT", FieldPath: "DisabledAt"},
		{Header: "CREATED BY", FieldPath: "CreatedBy"},
	}
	PrintAsTable(versionSets, columns)
}
