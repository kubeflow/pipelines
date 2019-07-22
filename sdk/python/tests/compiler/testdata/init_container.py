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

import kfp.dsl as dsl
from kubernetes import client as k8s_client


@dsl.pipeline(name='InitContainer', description='A pipeline with init container.')
def init_container_pipeline():

  echo = dsl.InitContainer(
      name='echo',
      image='alpine:latest',
      command=['echo', 'bye'])

  op1 = dsl.ContainerOp(
      name='hello',
      image='alpine:latest',
      command=['echo', 'hello'],
      init_containers=[echo])
