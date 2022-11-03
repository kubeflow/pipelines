#!/bin/bash

# Copyright 2018-2020 The Kubeflow Authors
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

set -ex

export TMP_OUTPUT=/tmp
API_VERSION=$API_VERSION
echo ${PROTOCCOMPILER}

# Change directory.
cd /go/src/github.com/kubeflow/pipelines
KFP_VERSION=$(cat VERSION)

# Delete currently generated code.
rm -r -f backend/api/${API_VERSION}/go_http_client/*
rm -f -f backend/api/${API_VERSION}/go_client/*
# Cannot delete backend/api/${API_VERSION}/swagger/*, because there are manually maintained definition files too.

# Create directories if they don't exist
mkdir -p backend/api/${API_VERSION}/go_http_client
mkdir -p backend/api/${API_VERSION}/go_client
mkdir -p backend/api/${API_VERSION}/swagger

# Generate *.pb.go (grpc api client) from *.proto.
${PROTOCCOMPILER} -I. -Ibackend/api/${API_VERSION} \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/ \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options/ \
    -I/usr/include/ \
    --plugin=protoc-gen-go=/go/bin/protoc-gen-go  \
    --go_out=plugins=grpc:${TMP_OUTPUT} \
    backend/api/${API_VERSION}/*.proto
# Generate *.pb.gw.go (grpc api rest client) from *.proto.
${PROTOCCOMPILER} -I. -Ibackend/api/${API_VERSION} \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/ \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options/ \
    -I/usr/include/ \
    --plugin=protoc-gen-grpc-gateway=/go/bin/protoc-gen-grpc-gateway \
    --grpc-gateway_out=logtostderr=true:${TMP_OUTPUT} \
    backend/api/${API_VERSION}/*.proto
# Move *.pb.go and *.gw.go to go_client folder.
cp ${TMP_OUTPUT}/github.com/kubeflow/pipelines/backend/api/${API_VERSION}/go_client/* ./backend/api/${API_VERSION}/go_client
# Generate *.swagger.json from *.proto into swagger folder.
${PROTOCCOMPILER} -I. -Ibackend/api/${API_VERSION} \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/ \
    -I/go/src/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger/options/ \
    -I//usr/include/ \
    --plugin=protoc-gen-swagger=/go/bin/protoc-gen-swagger \
    --swagger_out=logtostderr=true:${TMP_OUTPUT} \
    backend/api/${API_VERSION}/*.proto
# Move *.swagger.json files into swagger folder.
cp -a ${TMP_OUTPUT}/backend/api/${API_VERSION}/*.swagger.json ./backend/api/${API_VERSION}/swagger
# Generate a single swagger json file from the swagger json files of all models.
# Note: use backend/backend/api/${API_VERSION}/swagger/{run,job,experiment,pipeline,pipeline.upload,healthz}.swagger.json when apt-get can install jq-1.6
jq -s 'reduce .[] as $item ({}; . * $item) | .info.title = "Kubeflow Pipelines API" | .info.description = "This file contains REST API specification for Kubeflow Pipelines. The file is autogenerated from the swagger definition." | .info.version = "'$KFP_VERSION'" | .info.contact = { "name": "google", "email": "kubeflow-pipelines@google.com", "url": "https://www.google.com" } | .info.license = { "name": "Apache 2.0", "url": "https://raw.githubusercontent.com/kubeflow/pipelines/master/LICENSE" }' \
    backend/api/${API_VERSION}/swagger/experiment.swagger.json \
    backend/api/${API_VERSION}/swagger/run.swagger.json \
    backend/api/${API_VERSION}/swagger/job.swagger.json \
    backend/api/${API_VERSION}/swagger/pipeline.swagger.json \
    backend/api/${API_VERSION}/swagger/pipeline.upload.swagger.json \
    backend/api/${API_VERSION}/swagger/healthz.swagger.json \
    > "backend/api/${API_VERSION}/swagger/kfp_api_single_file.swagger.json"
# Generate go_http_client from swagger json.
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/job.swagger.json \
    -A job \
    --principal models.Principal \
    -c job_client \
    -m job_model \
    -t backend/api/${API_VERSION}/go_http_client
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/run.swagger.json \
    -A run \
    --principal models.Principal \
    -c run_client \
    -m run_model \
    -t backend/api/${API_VERSION}/go_http_client
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/experiment.swagger.json \
    -A experiment \
    --principal models.Principal \
    -c experiment_client \
    -m experiment_model \
    -t backend/api/${API_VERSION}/go_http_client
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/pipeline.swagger.json \
    -A pipeline \
    --principal models.Principal \
    -c pipeline_client \
    -m pipeline_model \
    -t backend/api/${API_VERSION}/go_http_client
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/pipeline.upload.swagger.json \
    -A pipeline_upload \
    --principal models.Principal \
    -c pipeline_upload_client \
    -m pipeline_upload_model \
    -t backend/api/${API_VERSION}/go_http_client
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/visualization.swagger.json \
    -A visualization \
    --principal models.Principal \
    -c visualization_client \
    -m visualization_model \
    -t backend/api/${API_VERSION}/go_http_client
swagger generate client \
    -f backend/api/${API_VERSION}/swagger/healthz.swagger.json \
    -A healthz \
    --principal models.Principal \
    -c healthz_client \
    -m healthz_model \
    -t backend/api/${API_VERSION}/go_http_client
# Hack to fix an issue with go-swagger
# See https://github.com/go-swagger/go-swagger/issues/1381 for details.
sed -i -- 's/MaxConcurrency int64 `json:"max_concurrency,omitempty"`/MaxConcurrency int64 `json:"max_concurrency,omitempty,string"`/g' backend/api/${API_VERSION}/go_http_client/job_model/${API_VERSION}_job.go
sed -i -- 's/IntervalSecond int64 `json:"interval_second,omitempty"`/IntervalSecond int64 `json:"interval_second,omitempty,string"`/g' backend/api/${API_VERSION}/go_http_client/job_model/${API_VERSION}_periodic_schedule.go
sed -i -- 's/MaxConcurrency string `json:"max_concurrency,omitempty"`/MaxConcurrency int64 `json:"max_concurrency,omitempty,string"`/g' backend/api/${API_VERSION}/go_http_client/job_model/${API_VERSION}_job.go
sed -i -- 's/IntervalSecond string `json:"interval_second,omitempty"`/IntervalSecond int64 `json:"interval_second,omitempty,string"`/g' backend/api/${API_VERSION}/go_http_client/job_model/${API_VERSION}_periodic_schedule.go
# Execute the //go:generate directives in the generated code.
cd backend/api && go generate ./...
