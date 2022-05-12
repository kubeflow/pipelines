#!/bin/bash

set -ex

# Run the following command under /frontend to execute this file
# npm run build:pipeline-spec


# Convert buffer to runtime object using protoc
# Download google protos as dependencies.
mkdir -p ../api/kfp_pipeline_spec/google/rpc/
curl https://raw.githubusercontent.com/googleapis/googleapis/047d3a8ac7f75383855df0166144f891d7af08d9/google/rpc/status.proto -o ../api/kfp_pipeline_spec/google/rpc/status.proto 


protoc --plugin=./node_modules/.bin/protoc-gen-ts_proto \
       --js_out="import_style=commonjs,binary:src/generated/pipeline_spec" \
       --grpc-web_out="import_style=commonjs+dts,mode=grpcweb:src/generated/pipeline_spec" \
       --ts_proto_opt="esModuleInterop=true" \
       --ts_proto_out="./src/generated/pipeline_spec" \
       --proto_path="../api/kfp_pipeline_spec" \
       ../api/kfp_pipeline_spec/pipeline_spec.proto ../api/kfp_pipeline_spec/google/rpc/status.proto -I.

# ARCHIVE 
# The following commands are used for generating pipeline spec ts objects via protobuf.js:
# https://github.com/protobufjs/protobuf.js
# Because protobuf.js doesn't support processing for Protobuf.Value yet, we temporarily switched over to use ts-proto:
# https://github.com/stephenh/ts-proto
# Before confirming the ts-proto is useful for KFP use case, I keep the commands below in case we need to switch back to protobuf.js.

# protoc --js_out="import_style=commonjs,binary:src/generated/pipeline_spec" \
# --grpc-web_out="import_style=commonjs+dts,mode=grpcweb:src/generated/pipeline_spec" \
# --proto_path="../api/kfp_pipeline_spec" \
# ../api/kfp_pipeline_spec/pipeline_spec.proto \
# ../api/kfp_pipeline_spec/google/rpc/status.proto

# # Encode proto string to buffer using protobuf.js
# npx pbjs -t static-module -w commonjs -o src/generated/pipeline_spec/pbjs_ml_pipelines.js ../api/kfp_pipeline_spec/pipeline_spec.proto
# npx pbts -o src/generated/pipeline_spec/pbjs_ml_pipelines.d.ts src/generated/pipeline_spec/pbjs_ml_pipelines.js 

# # Explaination of protobufjs cli tool:
# # Install protobufjs-cli by using the main library
# # Command: npm install protobufjs --save --save-prefix=~
# # In the future, this cli will have its own distribution, and isolate from main library.
