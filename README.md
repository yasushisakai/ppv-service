
# Generate protobuf files

assumes `protoc`

```bash
mkdir -p gen/go

# generate go files
protoc -I proto --go_out=gen/go --go_opt=paths=source_relative --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative proto/v1/ppv.proto
```
