#!/bin/bash -e
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


# The script configures build environment for xgboost4j and distributed
# XGBoost training package.
# This needs to run on a debian GCE VM (debian 8) to match dataproc workers.
# Steps:
# 1. Create a GCE debian VM.
# 2. On the VM under ~ directory, run
#     xgb4j_build.sh gs://b/path/to/XGBoost*.scala gs://b/o/path/to/hold/package
# The generated package (jar) will be copied to gs://b/o/path/to/hold/package.


sudo apt-get update
sudo apt install -t jessie-backports build-essential git maven openjdk-8-jre-headless \
                    openjdk-8-jre openjdk-8-jdk-headless openjdk-8-jdk ca-certificates-java -y

wget --no-verbose http://www.cmake.org/files/v3.5/cmake-3.5.2.tar.gz
tar xf cmake-3.5.2.tar.gz
cd cmake-3.5.2
./configure
make
sudo make install
cd ..

export LD_LIBRARY_PATH=/usr/local/lib:$LD_LIBRARY_PATH

sudo git clone --recursive https://github.com/dmlc/xgboost
cd xgboost/
sudo chmod -R 777 .
sudo make -j4
sudo chmod -R 777 .

cd jvm-packages
gsutil cp $1 ./xgboost4j-example/src/main/scala/ml/dmlc/xgboost4j/scala/example/spark/
mvn -DskipTests=true package

gsutil cp xgboost4j-example/target/xgboost4j-example-0.8-SNAPSHOT-jar-with-dependencies.jar $2
rm -rf cmake-3.5.2
rm -rf xgboost

