module github.com/philslol/kritis3m_scalev2

go 1.23.5

require (
	github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl v1.0.3
	github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker v0.0.0-20250127171153-04e29aca11d6
	github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang v0.0.0-00010101000000-000000000000
	github.com/gofrs/uuid/v5 v5.3.1
	github.com/golang/protobuf v1.5.4
	github.com/jackc/pgx-gofrs-uuid v0.0.0-20230224015001-1d428863c2e2
	github.com/jackc/pgx/v5 v5.0.0-alpha.1.0.20220402194133-53ec52aa174c
	github.com/jagottsicher/termcolor v1.0.2
	github.com/rs/zerolog v1.33.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/viper v1.19.0
	google.golang.org/grpc v1.70.0
	google.golang.org/protobuf v1.35.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241202173237-19429a94021a // indirect
)

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/puddle v1.3.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.10.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	golang.org/x/crypto v0.30.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
)

replace github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl/lib => ./pkg/go-asl/lib

replace github.com/Laboratory-for-Safe-and-Secure-Systems/go-asl => ./pkg/go-asl

replace github.com/Laboratory-for-Safe-and-Secure-Systems/mqtt_broker => ./pkg/mqtt_broker

replace github.com/Laboratory-for-Safe-and-Secure-Systems/paho.mqtt.golang => ./pkg/paho.mqtt.golang
