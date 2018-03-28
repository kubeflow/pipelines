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

FROM gcr.io/tensorflow/tensorflow:1.5.0

RUN pip install pyyaml==3.12 six==1.11.0

# TODO: Due to a pickling bug that causes image preprocessing failure we have to use HEAD.
# Install tft 0.6 when it is released.
RUN pip install https://storage.googleapis.com/ml-pipeline-playground/tensorflow-transform-0.6.0.dev0.tar.gz

ADD build /ml
WORKDIR /ml
