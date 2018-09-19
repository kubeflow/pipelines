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

repo_test_dir=$(dirname $0)

"${repo_test_dir}/install_docker.sh"
MINIKUBE_VERSION=v0.28.2 KUBECTL_VERSION=v1.11.2 "${repo_test_dir}/install_and_start_minikube_without_vm.sh"
ARGO_VERSION=v2.2.0 "${repo_test_dir}/install_argo_client.sh"
sudo apt-get install socat #needed for port forwarding
