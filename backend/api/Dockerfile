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
FROM golang:1.27.0@sha256:65b6f280bf050ec5af12716857e8ea8439d694dbba8f31ceeb7630670071f2bb AS generator
ENV PROTOC_VERSION=31.1
ENV GOBIN=/go/bin
ARG TARGETARCH
# The googleapis repo doesn't use GitHub releases or version tags,
# so we pin a specific commit to make the clone reproducible.
ENV GOOGLEAPIS_COMMIT=68d5196a529174df97c28c70622ffc1c3721815f

# Runtime-coupled Go generators use the exact versions selected by the root
# module. Standalone generator tools use the dedicated tools module so
# Dependabot can update them without adding build-only dependencies to KFP.
COPY go.mod /tmp/kfp-module/go.mod
COPY backend/api/tools/go.mod /tmp/api-generator-tools/go.mod

# Install protoc and Java (needed for KFP server API package generation).
RUN apt-get update -y && apt-get install -y jq sed unzip default-jdk
RUN set -eu; \
    if [ "$TARGETARCH" = "ppc64le" ]; then \
        PROTOC_ARCH=ppcle_64; \
    else \
        PROTOC_ARCH=x86_64; \
    fi; \
    curl -L -o protoc.zip "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-${PROTOC_ARCH}.zip"
RUN unzip -o protoc.zip -d /tmp/protoc && \
    mv /tmp/protoc/bin/protoc /usr/bin/protoc && \
    chmod +x /usr/bin/protoc
RUN unzip -o protoc.zip 'include/*' -d /tmp/protoc && \
    mv /tmp/protoc/include/* /usr/include
RUN rm -f protoc.zip
ENV PROTOCCOMPILER=/usr/bin/protoc
ENV PROTOCINCLUDE=/usr/include/google/protobuf

# Install the Go-based generators at their manifest-selected versions. Keep the
# grpc-gateway options alongside the plugins because protoc imports them.
RUN set -eu; \
    module_version() { cd "$1" && go mod edit -json | jq -er --arg module "$2" '.Require[] | select(.Path == $module) | .Version'; }; \
    grpc_gateway_version="$(module_version /tmp/kfp-module github.com/grpc-ecosystem/grpc-gateway/v2)"; \
    protobuf_go_version="$(module_version /tmp/kfp-module google.golang.org/protobuf)"; \
    protoc_gen_go_grpc_version="$(module_version /tmp/kfp-module google.golang.org/grpc/cmd/protoc-gen-go-grpc)"; \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@"${grpc_gateway_version}"; \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@"${grpc_gateway_version}"; \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@"${protobuf_go_version}"; \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@"${protoc_gen_go_grpc_version}"; \
    mkdir -p /protoc-gen-openapiv2; \
    cp -r "/go/pkg/mod/github.com/grpc-ecosystem/grpc-gateway/v2@${grpc_gateway_version}/protoc-gen-openapiv2/options" /protoc-gen-openapiv2/options

# Need to explicitly provide the googleapis protos and the OpenAPI options that were previously present in the grpc-gateway repo.
RUN git init /googleapis && \
    cd /googleapis && \
    git remote add origin https://github.com/googleapis/googleapis.git && \
    git fetch --depth 1 origin ${GOOGLEAPIS_COMMIT} && \
    git checkout FETCH_HEAD
# Download go-swagger binary.
RUN set -eu; \
    if [ "$TARGETARCH" = "ppc64le" ]; then \
        SWAGGER_ARCH=ppc64le; \
    else \
        SWAGGER_ARCH=amd64; \
    fi; \
    go_swagger_version="$(cd /tmp/api-generator-tools && go mod edit -json | jq -er '.Require[] | select(.Path == "github.com/go-swagger/go-swagger") | .Version')"; \
    curl -fL -o /usr/bin/swagger "https://github.com/go-swagger/go-swagger/releases/download/${go_swagger_version}/swagger_linux_amd64"; \
    chmod +x /usr/bin/swagger

# Needed for buildling python packages requiring protoc
RUN apt-get update && apt-get install -y python3-pip
RUN python3 -m pip install --no-cache-dir --break-system-packages --upgrade --ignore-installed pip setuptools wheel

# WORKAROUND: https://github.com/docker-library/golang/issues/225#issuecomment-403170792
ENV XDG_CACHE_HOME=/tmp/.cache
# Make all files accessible to non-root users.
RUN chmod -R 777 /usr/bin/
RUN chmod -R 777 /usr/include/google
RUN chmod -R 777 /go
