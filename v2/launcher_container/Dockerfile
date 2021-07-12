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

# 1. Build launcher binary
# Pin golang to 1.15 because of https://github.com/golang/go/issues/44129.
# Remove 1.15 pin when the issue gets resolved
FROM golang:1.15-alpine as builder

# Download go modules in a separate step for better docker caching.
WORKDIR /build
# api package is a dependency of v2/go.mod
COPY api/go.mod api/go.sum api/
COPY v2/go.mod v2/go.sum v2/
RUN cd v2 && go mod download
# Build the application
COPY api api
COPY v2 v2
WORKDIR /build/v2
RUN GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    go build -tags netgo -ldflags '-extldflags "-static"'  github.com/kubeflow/pipelines/v2/cmd/launch

# 2. Copy built binary into a minimal image.
FROM gcr.io/google-containers/busybox:1.27.2
# Include copyright notices and licenses, needs update each time dependencies change
COPY v2/third_party/NOTICES /NOTICES
WORKDIR /bin
COPY v2/launcher_container/mount_launcher.sh .
COPY --from=builder /build/v2/launch .
