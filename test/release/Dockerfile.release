# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# based on KFP backend api client generator dockerfile
FROM gcr.io/ml-pipeline-test/api-generator:latest

# install nvm & node 12
# Reference: https://stackoverflow.com/a/28390848
ENV NODE_VERSION 12.21.0
ENV NVM_DIR=/usr/local/nvm
RUN mkdir -p $NVM_DIR && \
    curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.37.2/install.sh | bash && \
    . $NVM_DIR/nvm.sh && \
    nvm install $NODE_VERSION && \
    nvm alias default $NODE_VERSION && \
    nvm use default
ENV NODE_PATH $NVM_DIR/versions/node/v$NODE_VERSION/lib/node_modules
ENV PATH $NVM_DIR/versions/node/v$NODE_VERSION/bin:$PATH

# install java==8 python==3
# adoptopenjdk apt repo is needed on debian buster, refer to https://stackoverflow.com/a/59436618/8745218
RUN apt-get install -y software-properties-common \
    && wget -qO - https://adoptopenjdk.jfrog.io/adoptopenjdk/api/gpg/key/public | apt-key add - \
    && add-apt-repository --yes https://adoptopenjdk.jfrog.io/adoptopenjdk/deb/ \
    && apt-get update \
    && apt-get install -y adoptopenjdk-8-hotspot python3-pip \
    && rm -rf /var/lib/apt/lists/*

# install setuptools
RUN python3 -m pip install setuptools

# install yq==3
# Released in https://github.com/mikefarah/yq/releases/tag/3.4.1
RUN curl -L -o /usr/local/bin/yq https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_amd64 && \
    chmod +x /usr/local/bin/yq

# Make all files accessible to non-root users.
RUN chmod -R 777 $NVM_DIR && \
    chmod -R 777 /usr/local/bin

# Configure npm cache location
RUN npm config set cache /tmp/.npm --global
