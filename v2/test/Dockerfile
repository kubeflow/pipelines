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

### Build backend compiler binary ###
# Pin golang to 1.15 because of https://github.com/golang/go/issues/44129.
# Remove 1.15 pin when the issue gets resolved
FROM golang:1.15-alpine as builder

# Install tools
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
RUN make build/compiler

FROM python:3.7-slim

WORKDIR /workdir
COPY v2/test/requirements.txt v2/test/
COPY sdk/python sdk/python
# relative path in requirements.txt are relative to workdir, so we need to
# cd to that folder first
RUN cd v2/test && pip3 install --no-cache-dir -r requirements.txt
COPY --from=builder /build/v2/build/compiler /bin/kfp-v2-compiler
# copy all other code
COPY . .
