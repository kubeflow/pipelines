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


FROM public.ecr.aws/amazonlinux/amazonlinux:2

ENV PYTHONPATH /app

RUN yum update -y \
    && yum install -y -q \
    ca-certificates \
    python3 \
    wget \
    unzip

RUN pip3 install uv \
    && uv pip install --system \
      "boto3==1.14.12" \
      "sagemaker==2.237.3" \
      "pathlib2==2.3.5" \
      "pyyaml==5.4" \
      "mypy-extensions==0.4.3" \
      "protobuf==3.20.*"

WORKDIR /app

COPY . .

ENTRYPOINT [ "bash" ]
