# create directory for proto generated files

echo ""
echo "generating protobuf"
echo ""
mkdir -p gen/go

# build protobuf
protoc -I proto -I /usr/local/include --go_out=gen/go --go_opt=paths=source_relative --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative proto/ppv/v1/ppv.proto

echo ""
echo "building the server"
echo ""

# build server
go build -o bin/ppv-server ./cmd/ppv-server
