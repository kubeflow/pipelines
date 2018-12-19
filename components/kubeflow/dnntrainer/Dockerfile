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

ARG TF_TAG
FROM tensorflow/tensorflow:$TF_TAG

RUN apt-get update -y

RUN apt-get install --no-install-recommends -y -q ca-certificates python-dev python-setuptools \
                                                  wget unzip git

RUN apt-get install --no-install-recommends -y -q build-essential && \
    pip install pyyaml==3.12 six==1.11.0 \
        tensorflow-transform==0.6.0 \
        tensorflow-model-analysis==0.6.0 && \
    apt-get --purge autoremove -y build-essential

ADD build /ml

RUN mkdir /usr/licenses && \
    /ml/license.sh /ml/third_party_licenses.csv /usr/licenses

WORKDIR /ml

ENTRYPOINT ["python", "-m", "trainer.task"]
