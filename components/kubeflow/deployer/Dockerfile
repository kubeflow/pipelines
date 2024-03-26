# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM debian

RUN apt-get update -q && apt-get upgrade -y && \
    apt-get install -y -qq --no-install-recommends \
      apt-transport-https \
      ca-certificates \
      git \
      gnupg \
      lsb-release \
      unzip \
      build-essential \
      wget && \
    wget -q https://go.dev/dl/go1.16.linux-s390x.tar.gz && \
    tar -xzf go1.16.linux-s390x.tar.gz && \
    mv go /usr/local && \
    rm -rf go1.16.linux-s390x.tar.gz && \
    wget -q -O /bin/kubectl \
      https://storage.googleapis.com/kubernetes-release/release/v1.11.2/bin/linux/s390x/kubectl && \
    chmod +x /bin/kubectl && \
    wget -q -O /opt/kubernetes_v1.11.2.tar.gz \
      https://github.com/kubernetes/kubernetes/archive/v1.11.2.tar.gz && \
    mkdir -p /src && \
    tar -C /src -xzf /opt/kubernetes_v1.11.2.tar.gz && \
    rm -rf /opt/kubernetes_v1.11.2.tar.gz

ENV GOROOT /usr/local/go
ENV GOPATH /go
ENV PATH $GOPATH/bin:$GOROOT/bin:$PATH

RUN go version

ENV KUBEFLOW_VERSION v0.4.0

# Checkout the kubeflow packages at image build time so that we do not
# require calling in to the GitHub API at run time.
RUN cd /src && \
    mkdir -p github.com/kubeflow && \
    cd github.com/kubeflow && \
    git clone https://github.com/kubeflow/kubeflow && \
    cd kubeflow && \
    git checkout ${KUBEFLOW_VERSION}

ADD ./src/deploy.sh /bin/.

ENTRYPOINT ["/bin/deploy.sh"]
