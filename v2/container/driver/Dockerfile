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

### Build driver binary ###
# Pin golang to 1.15 because of https://github.com/golang/go/issues/44129.
# Remove 1.15 pin when the issue gets resolved
FROM golang:1.15-alpine as builder

# Install tools
# TODO(Bobgy): integrate license checking
RUN apk add --no-cache make bash

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
RUN make build/driver

### Copy built binary into a minimal image ###
FROM scratch
WORKDIR /bin
COPY --from=builder /build/v2/build/driver ./kfp-driver
