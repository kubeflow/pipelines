#!/bin/bash

set -ex

# Run the following command under /frontend to execute this file
# npm run build:kubernetes-platform-spec

protoc --plugin=./node_modules/.bin/protoc-gen-ts_proto \
       --js_out="import_style=commonjs,binary:src/generated/platform_spec/kubernetes_platform" \
       --grpc-web_out="import_style=commonjs+dts,mode=grpcweb:src/generated/platform_spec/kubernetes_platform" \
       --ts_proto_opt="esModuleInterop=true" \
       --ts_proto_out="./src/generated/platform_spec/kubernetes_platform" \
       --proto_path="../kubernetes_platform/proto" \
       ../kubernetes_platform/proto/kubernetes_executor_config.proto -I.
