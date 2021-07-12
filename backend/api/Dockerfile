# Generate client code (go & json) from API protocol buffers
FROM golang:1.13.1-stretch as generator
ENV GRPC_GATEWAY_VERSION v1.9.0
ENV GO_SWAGGER_VERSION v0.18.0
ENV GOLANG_PROTOBUF_VERSION v1.3.2
ENV GRPC_VERSION v1.23.0
ENV PROTOC_VERSION 3.13.0
ENV GOBIN=/go/bin

# Install protoc.
RUN apt-get update -y && apt-get install -y jq sed unzip
RUN curl -L -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip -o protoc.zip -d /usr/ bin/protoc
RUN unzip -o protoc.zip -d /usr/ 'include/*'
RUN rm -f protoc.zip
ENV PROTOCCOMPILER /usr/bin/protoc
ENV PROTOCINCLUDE /usr/include/google/protobuf

# Need grpc-gateway source code for -I in protoc command.
WORKDIR /go/src/github.com
RUN mkdir grpc-ecosystem && cd grpc-ecosystem && git clone --depth 1 --branch $GRPC_GATEWAY_VERSION https://github.com/grpc-ecosystem/grpc-gateway.git
RUN mkdir grpc && git clone --depth 1 --branch $GRPC_VERSION https://github.com/grpc/grpc-go

# Install protoc-gen-rpc-gateway && protoc-gen-swagger.
RUN cd grpc-ecosystem/grpc-gateway && GO111MODULE=on go mod vendor
RUN go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
RUN go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

# Download go-swagger binary.
RUN curl -LO "https://github.com/go-swagger/go-swagger/releases/download/${GO_SWAGGER_VERSION}/swagger_linux_amd64"
RUN chmod +x swagger_linux_amd64 && mv swagger_linux_amd64 /usr/bin/swagger

# Need protobuf source code for -I in protoc command.
RUN mkdir golang && cd golang && git clone --depth 1 --branch $GOLANG_PROTOBUF_VERSION https://github.com/golang/protobuf.git
# Install protoc-gen-go.
RUN cd golang/protobuf && GO111MODULE=on go mod vendor
RUN go install github.com/golang/protobuf/protoc-gen-go

# WORKAROUND: https://github.com/docker-library/golang/issues/225#issuecomment-403170792
ENV XDG_CACHE_HOME /tmp/.cache
# Make all files accessible to non-root users.
RUN chmod -R 777 /usr/bin/
RUN chmod -R 777 /usr/include/google
RUN chmod -R 777 /go
