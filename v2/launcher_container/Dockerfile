# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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

WORKDIR /bin
RUN cp /build/launch .

ENTRYPOINT ["/bin/launch", "--mlmd_server_address", "${METADATA_GRPC_SERVICE_HOST}", "--mlmd_server_port", "${METADATA_GRPC_SERVICE_PORT}"]
