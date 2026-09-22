# Copyright 2021 The Kubeflow Authors
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

FROM python:3.11-slim

WORKDIR /workdir
RUN apt-get update && apt-get install --no-install-recommends -y make git wget protobuf-compiler \
    && rm -rf /var/lib/apt/lists
RUN pip install uv
COPY . .
RUN make -C api fetch-protos \
    && python3 api/v2alpha1/python/generate_proto.py \
    && python3 kubernetes_platform/python/generate_proto.py \
    && uv sync --frozen --extra backend-v2-test
ENV PATH="/workdir/.venv/bin:$PATH"
