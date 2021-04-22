# Pin golang to 1.15 because of https://github.com/golang/go/issues/44129.
# Remove 1.15 pin when the issue gets resolved
FROM golang:1.15-alpine

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Copy the code into the container
COPY . /build/.

# Build the application
WORKDIR /build
RUN go build -tags netgo -ldflags '-extldflags "-static"'  github.com/kubeflow/pipelines/v2/cmd/launch

COPY launcher_container/mount_launcher.sh /bin/mount_launcher.sh
RUN chmod +x /bin/mount_launcher.sh

WORKDIR /bin
RUN cp /build/launch .

ENTRYPOINT ["/bin/launch", "--mlmd_server_address", "${METADATA_GRPC_SERVICE_HOST}", "--mlmd_server_port", "${METADATA_GRPC_SERVICE_PORT}"]
