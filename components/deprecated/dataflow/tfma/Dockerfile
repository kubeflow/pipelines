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

FROM ml-pipeline-dataflow-base

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q build-essential && \
    pip install tensorflow-model-analysis==0.6.0 && \
    pip install ipywidgets==7.2.1 && \
    apt-get --purge autoremove -y build-essential

WORKDIR /ml

RUN mkdir /usr/licenses && \
    /ml/license.sh /ml/third_party_licenses.csv /usr/licenses

ENTRYPOINT ["python", "/ml/model_analysis.py"]
