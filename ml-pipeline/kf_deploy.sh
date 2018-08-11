#!/bin/bash

# Copyright 2018 Google LLC
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
#
# This file adds pipelines to an existing ksonnet application.
# It should be executed from a ksonnet directory.

set -xe

REGISTRY="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null && pwd )"

ks registry add ml-pipeline "${REGISTRY}"
ks pkg install ml-pipeline/ml-pipeline
ks generate ml-pipeline ml-pipeline
