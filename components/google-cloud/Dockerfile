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
FROM marketplace.gcr.io/google/ubuntu2404:latest
RUN apt update && apt -y install git python3 \
  python3-pip \
  python3-setuptools

WORKDIR /root

ENV PIP_BREAK_SYSTEM_PACKAGES=1

# Required by gcp_launcher
# Using google-cloud-aiplatform>=1.21.0 to avoid dataset creatation timeout
RUN pip3 install -U "google-cloud-aiplatform>=1.21.0"
RUN pip3 install -U google-cloud-storage
RUN pip3 install -U google-api-python-client

# Required by dataflow_launcher
RUN pip3 install -U apache_beam

# Required for sklearn/train_test_split_jsonl
RUN pip3 install -U "fsspec>=0.7.4" "gcsfs>=0.6.0" "pandas<=1.3.5" "scikit-learn"

# Required by experimental.notebooks.NotebooksExecutorOp
RUN pip3 install -U google-cloud-notebooks

# Install main package
RUN pip3 install "git+https://github.com/kubeflow/pipelines.git@google-cloud-pipeline-components-2.17.0#egg=google-cloud-pipeline-components&subdirectory=components/google-cloud"

# Note that components can override the container entry ponint.
ENTRYPOINT ["python3","-m","google_cloud_pipeline_components.container.v1.aiplatform.remote_runner"]
