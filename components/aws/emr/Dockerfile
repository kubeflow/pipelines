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

RUN pip install boto3==1.9.130 pathlib2

COPY create_cluster/src/create_cluster.py .
COPY delete_cluster/src/delete_cluster.py .
COPY submit_pyspark_job/src/submit_pyspark_job.py .
COPY submit_spark_job/src/submit_spark_job.py .
COPY common /app/common/

ENV PYTHONPATH /app

ENTRYPOINT [ "bash" ]
