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

FROM python:3.7-slim

WORKDIR /workdir
COPY v2/test/requirements.txt v2/test/
COPY v2/test/requirements-runner.txt v2/test/
COPY sdk/python sdk/python
COPY samples/test/utils samples/test/utils
# relative path in requirements.txt are relative to workdir, so we need to
# cd to that folder first
RUN cd v2/test && pip3 install --no-cache-dir -r requirements-runner.txt
# copy all other code
COPY . .
