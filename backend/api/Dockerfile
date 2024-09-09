# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Generate client code (go & json) from API protocol buffers
FROM golang:1.21 as generator
ENV GRPC_GATEWAY_VERSION v1.9.6
ENV GO_SWAGGER_VERSION v0.18.0
ENV GOLANG_PROTOBUF_VERSION v1.5.1
ENV GRPC_VERSION v1.23.0
ENV PROTOC_VERSION 3.17.3
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
RUN go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@latest
RUN go install github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@latest

# Download go-swagger binary.
RUN curl -LO "https://github.com/go-swagger/go-swagger/releases/download/${GO_SWAGGER_VERSION}/swagger_linux_amd64"
RUN chmod +x swagger_linux_amd64 && mv swagger_linux_amd64 /usr/bin/swagger

# Need protobuf source code for -I in protoc command.
RUN mkdir golang && cd golang && git clone --depth 1 --branch $GOLANG_PROTOBUF_VERSION https://github.com/golang/protobuf.git
# Install protoc-gen-go.
RUN cd golang/protobuf && GO111MODULE=on go mod vendor
RUN go install github.com/golang/protobuf/protoc-gen-go@latest

# WORKAROUND: https://github.com/docker-library/golang/issues/225#issuecomment-403170792
ENV XDG_CACHE_HOME /tmp/.cache
# Make all files accessible to non-root users.
RUN chmod -R 777 /usr/bin/
RUN chmod -R 777 /usr/include/google
RUN chmod -R 777 /go
