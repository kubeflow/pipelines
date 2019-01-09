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

FROM python:3.6-alpine

RUN apk update && apk add --no-cache bash libffi-dev gcc musl-dev libressl-dev

RUN pip install --no-cache-dir -U kubernetes==8.0.1 google-api-python-client==1.6.2 fire==0.1.3 oauth2client==3.0.0 google-cloud-storage==1.13.2

ADD build /ml

RUN mkdir /usr/licenses && \
    /ml/license.sh /ml/third_party_licenses.csv /usr/licenses

WORKDIR /ml
ENTRYPOINT ["python", "-u", "-m"]
