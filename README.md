
# Generate protobuf files

assumes `protoc`

```bash
mkdir -p gen/go

# generate go files
protoc -I proto --go_out=gen/go --go_opt=paths=source_relative --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative proto/v1/ppv.proto
```

## How to use simple client

This submits and waits for the result. Here are all the options with the defaults:

```bash
./bin/ppv-client --address=localhost \
                 --port=50051 \
                 --matrix="0.0,0.1,0.0,0.0,0.2,0.0,0.0,0.0,0.8,0.0,1.0,0.0,0.0,0.9,0.0,1.0" \
                 --delegates=2 \
                 --policies=2 \
                 --intermediaries=0
```

## Docker 

### Building the server Docker image

This assumes you have access to the ppv library

```bash
docker build -t ppv-service .
```

### Running the Docker container

```bash
docker run -d -p 50051:50051 --name ppv-container ppv-service
```
