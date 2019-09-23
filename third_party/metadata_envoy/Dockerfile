# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM envoyproxy/envoy:latest

RUN apt-get update -y && \
  apt-get install --no-install-recommends -y -q python python-pip python-setuptools gettext
RUN pip install wget

COPY third_party/metadata_envoy/envoy.yaml /etc/envoy.yaml

# Copy license files.
ADD third_party/metadata_envoy /scratch
RUN python /scratch/dependency_helper.py /scratch/dependencies.json
RUN rm -rf /scratch

RUN pip uninstall --yes wget
RUN apt-get remove --yes --purge python-setuptools  python-pip python

ENTRYPOINT ["/usr/local/bin/envoy", "-c"]
CMD ["/etc/envoy.yaml"]
