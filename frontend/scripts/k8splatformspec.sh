#!/bin/bash

set -ex

# Run the following command under /frontend to execute this file
# npm run build:platform-spec:kubernetes-platform

output_dir=${PROTO_OUT_DIR:-./src/generated/platform_spec/kubernetes_platform}
mkdir -p "$output_dir"

protoc --plugin=./node_modules/.bin/protoc-gen-ts_proto \
       --ts_proto_opt="esModuleInterop=true" \
       --ts_proto_out="$output_dir" \
       --proto_path="../kubernetes_platform/proto" \
       --proto_path="../api/v2alpha1" \
       ../kubernetes_platform/proto/kubernetes_executor_config.proto -I.
