```bash
protoc   --go_out=./gen/go/v1   --go_opt=paths=source_relative   --go-grpc_out=./gen/go/v1   --go-grpc_opt=paths=source_relative   -I ./proto   proto/*.proto --experimental_allow_proto3_optional
```

# Todo

- [] DB conn is not handled correcly, context is in new db is used for creation, but not passed, clients using the db should call conn close after usage
- [x] control plane does not use context yet