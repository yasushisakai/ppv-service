FROM golang:1.24 AS builder

# Install protobuf compiler and required libraries
RUN apt-get update && apt-get install -y \
    protobuf-compiler \
    build-essential \
    libblas-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Go protobuf plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /app

# Copy source code (including go.mod and local modules)
COPY . .

# Generate protobuf code
RUN mkdir -p gen/go && \
    protoc -I proto -I /usr/include --go_out=gen/go --go_opt=paths=source_relative --go-grpc_out=gen/go --go-grpc_opt=paths=source_relative proto/ppv/v1/ppv.proto

# Ensure libppv is built with the correct flags
RUN cd ppv/libppv && \
    make clean && \
    make && \
    cd ../..

# Build the Go service
RUN go mod download && \
    CGO_ENABLED=1 go build -o bin/ppv-server ./cmd/ppv-server

# Use the same base image for runtime to avoid GLIBC version issues
FROM golang:1.24

RUN apt-get update && apt-get install -y \
    libblas3 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Create the required directory structure
RUN mkdir -p /app/ppv/libppv

# Copy the binary and required libraries
COPY --from=builder /app/bin/ppv-server /app/
COPY --from=builder /app/ppv/libppv/libppv.a /app/ppv/libppv/

# Set the default port
EXPOSE 50051

# Command to run the service
CMD ["/app/ppv-server"]