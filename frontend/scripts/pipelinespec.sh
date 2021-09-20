#!/bin/bash

set -ex

# Run the following command under /frontend to execute this file
# npm run build:pipeline-spec


# Convert buffer to runtime object using protoc
# Download google protos as dependencies.
curl https://raw.githubusercontent.com/googleapis/googleapis/047d3a8ac7f75383855df0166144f891d7af08d9/google/rpc/status.proto -o ../api/v2alpha1/google/rpc/status.proto 

protoc --js_out="import_style=commonjs,binary:src/generated/pipeline_spec" \
--grpc-web_out="import_style=commonjs+dts,mode=grpcweb:src/generated/pipeline_spec" \
--proto_path="../api/v2alpha1" \
../api/v2alpha1/pipeline_spec.proto \
../api/v2alpha1/google/rpc/status.proto

# Encode proto string to buffer using protobuf.js
npx pbjs -t static-module -w commonjs -o src/generated/pipeline_spec/pbjs_ml_pipelines.js ../api/v2alpha1/pipeline_spec.proto
npx pbts -o src/generated/pipeline_spec/pbjs_ml_pipelines.d.ts src/generated/pipeline_spec/pbjs_ml_pipelines.js 

# Explaination of protobufjs cli tool:
# Install protobufjs-cli by using the main library
# Command: npm install protobufjs --save --save-prefix=~
# In the future, this cli will have its own distribution, and isolate from main library.
