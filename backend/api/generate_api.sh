#!/bin/bash

# Copyright 2018-2020 Google LLC
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


# This file generates API sources from the protocol buffers defined in this
# directory using Bazel, then copies them back into the source tree so they can
# be checked-in.

set -ex

DOCKERFILE="Dockerfile.buildapi"
IMAGE="builder"
RUNNER="dummy"
DIR="backend/api/"

## https://stackoverflow.com/questions/22049212/docker-copying-files-from-docker-container-to-host

cd ../..

# Delete currently generated code.
rm -r -f $DIR/go_http_client/*
rm -r -f $DIR/go_client/*

# Build the image, and generate the API:s
docker build . -f backend/$DOCKERFILE -t $IMAGE
# Create writable container layer
docker create -ti --name $RUNNER $IMAGE bash
# Copy the gnerate code
docker cp $RUNNER:/go/src/github.com/kubeflow/pipelines/backend/api/go_client/ ./backend/api/
docker cp $RUNNER:/go/src/github.com/kubeflow/pipelines/backend/api/go_http_client/ ./backend/api/
docker cp $RUNNER:/go/src/github.com/kubeflow/pipelines/backend/api/swagger/ ./backend/api/
docker rm -f $RUNNER