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

# The PWD is /home/prow/go/src/github.com/kubeflow/pipelines
# Navigate to /home/prow/
cd ../../../../..
mkdir go1.13.3
cd go1.13.3
apt-get update
apt-get -y upgrade
wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
# Go will be installed in /home/prow/go1.13.3/
tar -xvf go1.13.3.linux-amd64.tar.gz
# Naviage from /home/prow/go1.13.3 to project directory.
cd ../go/src/github.com/kubeflow/pipelines
../../../../../go1.13.3/go/bin/go mod vendor
../../../../../go1.13.3/go/bin/go test -v -cover ./backend/...
