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


@dsl.pipeline(name='Sidecar', description='A pipeline with sidecars.')
def sidecar_pipeline():

    echo = dsl.Sidecar(
        name='echo',
        image='hashicorp/http-echo',
        args=['-text="hello world"'])

    op1 = dsl.ContainerOp(
        name='download',
        image='busybox',
        command=['sh', '-c'],
        arguments=['sleep 10; wget localhost:5678 -O /tmp/results.txt'],
        sidecars=[echo],
        file_outputs={'downloaded': '/tmp/results.txt'})

    op2 = dsl.ContainerOp(
        name='echo',
        image='library/bash',
        command=['sh', '-c'],
        arguments=['echo %s' % op1.output])
