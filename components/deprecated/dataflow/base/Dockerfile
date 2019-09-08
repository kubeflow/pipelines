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

RUN apt-get update -y && apt-get install --no-install-recommends -y -q ca-certificates python-dev python-setuptools \
                                                  wget unzip git

RUN easy_install pip

RUN pip install absl-py==0.3.0 \
        apache-beam==2.5.0 \
        astor==0.7.1 \
        avro==1.8.2 \
        backports.weakref==1.0.post1 \
        bleach==1.5.0 \
        cachetools==2.1.0 \
        certifi==2018.4.16 \
        chardet==3.0.4 \
        crcmod==1.7 \
        dill==0.2.6 \
        docopt==0.6.2 \
        enum34==1.1.6 \
        fasteners==0.14.1 \
        funcsigs==1.0.2 \
        future==0.16.0 \
        futures==3.2.0 \
        gapic-google-cloud-pubsub-v1==0.15.4 \
        gast==0.2.0 \
        google-apitools==0.5.20 \
        google-auth==1.5.1 \
        google-auth-httplib2==0.0.3 \
        google-cloud-bigquery==0.25.0 \
        google-cloud-core==0.25.0 \
        google-cloud-pubsub==0.26.0 \
        google-gax==0.15.16 \
        googleapis-common-protos==1.5.3 \
        googledatastore==7.0.1 \
        grpc-google-iam-v1==0.11.4 \
        grpcio==1.14.0 \
        hdfs==2.1.0 \
        html5lib==0.9999999 \
        httplib2==0.9.2 \
        idna==2.7 \
        Markdown==2.6.11 \
        mock==2.0.0 \
        monotonic==1.5 \
        numpy==1.15.0 \
        oauth2client==4.1.2 \
        pbr==4.2.0 \
        ply==3.8 \
        proto-google-cloud-datastore-v1==0.90.4 \
        proto-google-cloud-pubsub-v1==0.15.4 \
        protobuf==3.6.0 \
        pyasn1==0.4.4 \
        pyasn1-modules==0.2.2 \
        pytz==2018.5 \
        PyVCF==0.6.8 \
        PyYAML==3.12 \
        requests==2.19.1 \
        rsa==3.4.2 \
        six==1.11.0 \
        tensorboard==1.6.0 \
        tensorflow==1.6.0 \
        tensorflow-transform==0.6.0 \
        termcolor==1.1.0 \
        typing==3.6.4 \
        urllib3==1.23 \
        Werkzeug==0.14.1

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

ADD build /ml

ENV PATH $PATH:/tools/node/bin:/tools/google-cloud-sdk/bin
