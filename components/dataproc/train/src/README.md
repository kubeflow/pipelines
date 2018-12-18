# XGBoost Distributed Trainer and Predictor Package


This directory contains code and building script to build a generic XGBoost model and perform
predictions on it.

XGBoost4j package currently requires building from source
(https://github.com/dmlc/xgboost/issues/1807), as well as the spark layer and user code on top
of it. To do so, get a GCE VM (debian 8), and run
[xgb4j_build.sh](xgb4j_build.sh) on it. The script contains steps to compile/install cmake,
git clone xgboost repo, copy sources, and build a jar package that can run in a spark environment.
This is only tested on Google Dataproc cluster.
