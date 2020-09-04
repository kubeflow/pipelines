#!/bin/bash -e
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


mkdir -p ./build/common
rsync -arvp "../analyze/src"/ ./build/
rsync -arvp "../train/src"/ ./build/
rsync -arvp "../predict/src"/ ./build/
rsync -arvp "../create_cluster/src"/ ./build/
rsync -arvp "../delete_cluster/src"/ ./build/
rsync -arvp "../transform/src"/ ./build/
rsync -arvp "../common"/ ./build/common/

docker build -t ml-pipeline-dataproc-base .
rm -rf ./build

