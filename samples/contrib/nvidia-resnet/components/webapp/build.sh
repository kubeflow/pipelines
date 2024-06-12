#!/bin/bash
# Copyright (c) 2019, NVIDIA CORPORATION.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

IMAGE=cculab509/webapp

# Build base TRTIS client image
base=tensorrt-inference-server
git clone https://github.com/triton-inference-server/client.git $base/clientrepo
docker build -t base-trtis-client -f $base/Dockerfile.sdk $base
rm -rf $base

# Build & push webapp image
docker build -t $IMAGE .
docker push $IMAGE
