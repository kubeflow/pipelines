#!/bin/bash
#
# Copyright 2020 Google LLC
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

# This bash launches diagnose steps.

project=''
cluster=''
region=''
namespace=''

print_usage() {
  printf "Usage: $0 -p project -c cluster -r region -n namespace\n"
}

while getopts ':p:c:r:n:' flag; do
  case "${flag}" in
    p) project=${OPTARG} ;;
    c) cluster=${OPTARG} ;;
    r) region="${OPTARG}" ;;
    n) namespace="${OPTARG}" ;;
    *) print_usage
       exit 1 ;;
  esac
done

if [ -z "${project}" ] || [ -z "${cluster}" ] || [ -z "${region}" ] || [ -z "${namespace}" ]; then
  print_usage
  exit 1
fi

echo "This tool helps you diagnose your KubeFlow Pipelines and provide configure suggestions."
echo "Start checking [project: ${project}, cluster: ${cluster}, region: ${region}, namespace: ${namespace}]"

# Do anything here

echo "Done"