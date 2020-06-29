#!/bin/bash -e
#
# Copyright 2019 Google LLC
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

# This script automated the process to release the component images.
# To run it, find a good release candidate commit SHA from ml-pipeline-staging project,
# and provide a full github COMMIT SHA to the script. E.g.
# ./release.sh 2118baf752d3d30a8e43141165e13573b20d85b8
# The script copies the images from staging to prod, and update the local code.
# You can then send a PR using your local branch.

cd "$(dirname "$0")"

PYTHONPATH="$PYTHONPATH:../sdk/python"

echo "Testing loading all components"
find . -name component.yaml | python3 -c '
import sys
import kfp
for component_file in sys.stdin:
  component_file = component_file.rstrip("\n")
  print(component_file)
  kfp.components.load_component_from_file(component_file)
'
