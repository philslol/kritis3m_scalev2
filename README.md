```bash
protoc   --go_out=./gen/go/v1   --go_opt=paths=source_relative   --go-grpc_out=./gen/go/v1   --go-grpc_opt=paths=source_relative   -I ./proto   proto/*.proto --experimental_allow_proto3_optional
```
