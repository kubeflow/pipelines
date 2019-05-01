#!/bin/bash -ex
#
# Copyright 2018 Google LLC
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


# The scripts creates the Kubeflow Pipelines python SDK package.
#
# Usage:
#   ./build.sh [output_file]


target_archive_file=${1:-kfp.tar.gz}

pushd "$(dirname "$0")"
dist_dir=$(mktemp -d)
python setup.py sdist --format=gztar --dist-dir "$dist_dir"
cp "$dist_dir"/*.tar.gz "$target_archive_file"
popd
