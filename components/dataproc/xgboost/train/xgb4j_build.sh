#!/bin/bash
#
# The script configures build environment for xgboost4j and distributed
# XGBoost training package.
# This needs to run on a debian GCE VM to match dataproc workers.
# Steps:
# 1. Create a GCE debian VM.
# 2. On the VM under ~ directory, run
#     xgb4j_build.sh gs://b/path/to/XGBoostTrainer.scala gs://b/o/path/to/hold/package
# The generated package (jar) will be copied to gs://b/o/path/to/hold/package.


sudo apt-get update
sudo apt-get install -y build-essential git maven openjdk-8-jre  openjdk-8-jdk-headless openjdk-8-jdk 


wget http://www.cmake.org/files/v3.5/cmake-3.5.2.tar.gz
tar xf cmake-3.5.2.tar.gz
cd cmake-3.5.2
./configure
make
sudo make install
cd ..

# export PATH=/usr/local/bin:$PATH
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

