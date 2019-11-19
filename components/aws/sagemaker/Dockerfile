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

RUN apt-get update -y && apt-get install --no-install-recommends -y -q ca-certificates python-dev python-setuptools wget unzip

RUN easy_install pip

RUN pip install boto3 sagemaker pathlib2 pyyaml==3.12

COPY hyperparameter_tuning/src/hyperparameter_tuning.py .
COPY train/src/train.py .
COPY deploy/src/deploy.py .
COPY model/src/create_model.py .
COPY batch_transform/src/batch_transform.py .
COPY workteam/src/workteam.py .
COPY ground_truth/src/ground_truth.py .

COPY common /app/common/

ENV PYTHONPATH /app

ENTRYPOINT [ "bash" ]
