# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the \"License\");
# you may not use this file except in compliance with the License.\n",
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an \"AS IS\" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Base image to use for this docker
FROM gcr.io/google-appengine/python

WORKDIR /root

# Upgrade pip to latest
RUN pip3 install --upgrade pip

# Installs additional packages
RUN pip3 install git+https://github.com/googleapis/python-aiplatform@mb-release

# Copies the trainer code to the docker image.
# TODO(chavoshi) change from using PR @531127d0f7af525976fae8581796a675d8703fbb version to master 
RUN pip3 install "git+https://github.com/kubeflow/pipelines.git@fcd24983168bca93d9e0c3ef91cafa46f3612977#egg=google-cloud-components&subdirectory=components/google-cloud"

# Sets up the entry point to invoke the trainer.
ENTRYPOINT ["python3","-m","google_cloud_components.mb_sdk_component.remote_runner"]