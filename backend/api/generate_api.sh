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

cd ../..
# Generate API
docker run  --interactive --rm \
    --mount type=bind,source="$(pwd)",target=/app/pipelines \
    builder /app/pipelines/backend/api/generator.sh

# Change owner to user for generate files, explenation of command: https://askubuntu.com/questions/829537/how-do-i-change-owner-to-current-user-on-folder-and-containing-folders-inside-my
sudo find backend/api -user root -exec sudo chown $USER: {} +