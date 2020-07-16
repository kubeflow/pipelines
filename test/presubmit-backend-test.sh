#!/bin/bash
#
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# usage: `./hack/run_unit_tests_backend.sh` to run backend unit tests once
#        `WATCH=true ./hack/run_unit_tests_backend.sh` to watch code changes and auto rerun tests
# Note: ibazel can be downloaded from https://github.com/bazelbuild/bazel-watcher

COMMAND="bazel"
if [ -n "$WATCH" ]; then
  COMMAND="ibazel"
fi
$COMMAND --host_jvm_args=-Xmx500m --host_jvm_args=-Xms500m test \
  --noshow_progress --noshow_loading_progress --define=grpc_no_ares=true \
  --test_output=all //backend/...
