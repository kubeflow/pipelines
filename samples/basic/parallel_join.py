# Copyright 2018 Google LLC
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


import mlp


@mlp.pipeline(
  name='Parallel_and_Join',
  description='Download two messages in parallel and print the concatenated result.'
)
def download_and_join(url1: mlp.PipelineParam, url2: mlp.PipelineParam):
  """A very simple two-step pipeline."""

  download1 = mlp.ContainerOp(
     name='download1',
     image='google/cloud-sdk',
     command=['sh', '-c'],
     arguments=['gsutil cat %s | tee /tmp/results.txt' % url1],
     file_outputs={'downloaded': '/tmp/results.txt'})

  download2 = mlp.ContainerOp(
     name='download2',
     image='google/cloud-sdk',
     command=['sh', '-c'],
     arguments=['gsutil cat %s | tee /tmp/results.txt' % url2],
     file_outputs={'downloaded': '/tmp/results.txt'})

  echo = mlp.ContainerOp(
     name='echo',
     image='library/bash',
     command=['sh', '-c'],
     arguments=['echo %s %s' % (download1.output, download2.output)])


