# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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

# Base image to use for this docker
FROM gcr.io/google-appengine/python:latest

WORKDIR /root

# Upgrade pip to latest
RUN pip3 install --upgrade pip

# Required by gcp_launcher
RUN pip3 install -U google-cloud-aiplatform
RUN pip3 install -U google-cloud-storage

# Required by dataflow_launcher
RUN pip3 install -U apache_beam[gcp]

# Install main package (switch to using pypi package for official release)
RUN pip3 install "git+https://github.com/kubeflow/pipelines.git#egg=google-cloud-pipeline-components&subdirectory=components/google-cloud"

# Note that components can override the container entry ponint.
ENTRYPOINT ["python3","-m","google_cloud_pipeline_components.container.aiplatform.remote_runner"]
