# Copyright 2019 The Kubeflow Authors
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

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q ca-certificates python-dev python-setuptools wget unzip git && \
    easy_install pip && \
    pip install pyyaml==3.12 six==1.11.0 requests==2.18.4 grpcio gcloud google-api-python-client protobuf kubernetes && \
    wget https://github.com/kubeflow/katib/archive/master.zip && unzip master.zip

ENV PYTHONPATH $PYTHONPATH:/katib-master/pkg/api/v1alpha1/python:/katib-master/py

ADD build /ml

RUN mkdir /usr/licenses && \
    /ml/license.sh /ml/third_party_licenses.csv /usr/licenses

ENTRYPOINT ["python", "/ml/launch_study_job.py"]
