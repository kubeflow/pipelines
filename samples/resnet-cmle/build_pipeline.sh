#!/bin/bash
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


BUILD_DIR=/tmp/build
rm -rf $BUILD_DIR
mkdir -p $BUILD_DIR
rsync -av ./preprocess $BUILD_DIR

pushd ./resnet_model
python setup.py sdist
rsync -av ./dist/ $BUILD_DIR/trainer/
rm -rf ./dist
popd

gsutil -m -h "Cache-Control:private, max-age=0, no-transform" rsync -rud $BUILD_DIR gs://ml-pipeline-playground/samples/ml_engine/resnet-cmle/ || true
rm -rf $BUILD_DIR

gsutil ls gs://ml-pipeline-playground/samples/ml_engine/resnet-cmle/

python ./resnet-train-pipeline.py