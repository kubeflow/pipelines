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

FROM ubuntu:16.04

RUN apt-get update -y

RUN apt-get install --no-install-recommends -y -q ca-certificates python-dev python-setuptools \
                                                  wget unzip git

RUN easy_install pip

RUN pip install pyyaml==3.12 six==1.11.0 tensorflow==1.8.0 apache-beam==2.5.0 

RUN pip install tensorflow-transform==0.6.0

RUN wget -nv https://dl.google.com/dl/cloudsdk/release/google-cloud-sdk.zip && \
    unzip -qq google-cloud-sdk.zip -d tools && \
    rm google-cloud-sdk.zip && \
    tools/google-cloud-sdk/install.sh --usage-reporting=false \
        --path-update=false --bash-completion=false \
        --disable-installation-options && \
    tools/google-cloud-sdk/bin/gcloud -q components update \
        gcloud core gsutil && \
    tools/google-cloud-sdk/bin/gcloud config set component_manager/disable_update_check true && \
    touch /tools/google-cloud-sdk/lib/third_party/google.py

RUN pip install --upgrade google-api-python-client

ENV PATH $PATH:/tools/node/bin:/tools/google-cloud-sdk/bin

ENV PYTHONPATH ${PYTHONPATH}:/resnet/resnet_model

ADD /build /resnet

RUN mkdir /usr/licenses && \
    /resnet/license.sh /resnet/third_party_licenses.csv /usr/licenses

ENTRYPOINT ["python", "/resnet/preprocess.py"]
