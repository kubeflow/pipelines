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

FROM python:2.7-slim-jessie

RUN apt-get update && apt-get install -y --no-install-recommends \
	wget patch \
	&& rm -rf /var/lib/apt/lists/*

RUN pip2 install apache-beam[gcp]==2.10.0
RUN pip install pandas

ADD build /ml
WORKDIR /ml
RUN pip install .

RUN patch /usr/local/lib/python2.7/site-packages/googleapiclient/http.py < /ml/patches/http.patch

ENTRYPOINT ["python", "-u", "-m", "kfp_component.launcher"]
