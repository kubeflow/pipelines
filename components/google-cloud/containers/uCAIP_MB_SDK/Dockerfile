# Copyright 2019 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the \"License\");
# you may not use this file except in compliance with the License.\n",
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an \"AS IS\" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Base image to use for this docker
FROM gcr.io/google-appengine/python

WORKDIR /root

# Installs additional packages
RUN pip3 install git+https://github.com/googleapis/python-aiplatform@mb-release

# Copies the trainer code to the docker image.
# COPY remote_runner.py /root/remote_runner.py

# Sets up the entry point to invoke the trainer.
# ENTRYPOINT ["python3", "remote_runner.py"]