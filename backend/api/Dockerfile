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
FROM golang:1.24 as generator
ENV GRPC_GATEWAY_VERSION v2.27.1
ENV GO_SWAGGER_VERSION v0.32.3
ENV GRPC_VERSION v1.73.0
ENV PROTOC_VERSION 31.1
ENV GOBIN=/go/bin
# The googleapis repo doesn't use GitHub releases or version tags,
# so we pin a specific commit to make the clone reproducible.
ENV GOOGLEAPIS_COMMIT 68d5196a529174df97c28c70622ffc1c3721815f

# **Note** that protoc-gen-go-grpc is packaged with grpc-go but is versioned
# separately. You can find the releases for protoc-gen-go-grpc here:
# https://github.com/grpc/grpc-go/releases
# **Note** that these also include releases for grpc-go which is the grpc Go
# runtime package. protoc-gen-go-grpc is the package used for generating
# Go GRPC code from .proto files.
# to list recent protoc-gen-go-grpc versions you can also do:
# go list -m -versions google.golang.org/grpc/cmd/protoc-gen-go-grpc
# PROTOC_GEN_GO_GRPC & PROTOBUF_GO versions should match reasonably close to each other.
# You can check the protobuf in the go.mod for protoc-gen-go-grpc, like here:
# https://github.com/grpc/grpc-go/blob/cmd/protoc-gen-go-grpc/v1.5.1/cmd/protoc-gen-go-grpc/go.mod#L7
# **Note** That BOTH PROTOC_GEN_GO_GRPC & PROTOBUF_GO here are used for
# Generating GO Code. These versions should be identical to the
# runtime Go packages (in the project go.mod)
ENV PROTOC_GEN_GO_GRPC v1.5.1
ENV PROTOBUF_GO=v1.36.6

# Install protoc.
RUN apt-get update -y && apt-get install -y jq sed unzip
RUN curl -L -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip -o protoc.zip -d /tmp/protoc && \
    mv /tmp/protoc/bin/protoc /usr/bin/protoc && \
    chmod +x /usr/bin/protoc
RUN unzip -o protoc.zip 'include/*' -d /tmp/protoc && \
    mv /tmp/protoc/include/* /usr/include
RUN rm -f protoc.zip
ENV PROTOCCOMPILER /usr/bin/protoc
ENV PROTOCINCLUDE /usr/include/google/protobuf

# Need grpc-gateway source code for -I in protoc command.
# Install protoc-gen-rpc-gateway && protoc-gen-swagger.
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@${GRPC_GATEWAY_VERSION}
RUN go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@${GRPC_GATEWAY_VERSION}

# Need to explicitly provide the googleapis protos and the OpenAPI options that were previously present in the grpc-gateway repo.
RUN git init /googleapis && \
    cd /googleapis && \
    git remote add origin https://github.com/googleapis/googleapis.git && \
    git fetch --depth 1 origin ${GOOGLEAPIS_COMMIT} && \
    git checkout FETCH_HEAD
RUN mkdir -p /protoc-gen-openapiv2 && \
    cp -r /go/pkg/mod/github.com/grpc-ecosystem/grpc-gateway/v2@${GRPC_GATEWAY_VERSION}/protoc-gen-openapiv2/options /protoc-gen-openapiv2/options

# Download go-swagger binary.
RUN curl -LO "https://github.com/go-swagger/go-swagger/releases/download/${GO_SWAGGER_VERSION}/swagger_linux_amd64"
RUN chmod +x swagger_linux_amd64 && mv swagger_linux_amd64 /usr/bin/swagger

# Need protobuf source code for -I in protoc command.
# Install protoc-gen-go.
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOBUF_GO}
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC}

# Needed for buildling python packages requiring protoc
RUN apt-get update && apt-get install -y python3-pip
RUN pip3 install --upgrade pip setuptools wheel --break-system-packages

# WORKAROUND: https://github.com/docker-library/golang/issues/225#issuecomment-403170792
ENV XDG_CACHE_HOME /tmp/.cache
# Make all files accessible to non-root users.
RUN chmod -R 777 /usr/bin/
RUN chmod -R 777 /usr/include/google
RUN chmod -R 777 /go
