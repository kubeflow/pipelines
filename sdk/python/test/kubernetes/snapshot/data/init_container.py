# Copyright 2026 The Kubeflow Authors
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

from kfp import dsl
from kfp.kubernetes import add_init_container
from kfp.kubernetes import empty_dir_mount


@dsl.component
def print_greeting():
    print('hello world')


@dsl.pipeline
def my_pipeline():
    task = print_greeting()
    empty_dir_mount(
        task,
        volume_name='shared-data',
        mount_path='/data',
    )
    add_init_container(
        task,
        name='prepare-data',
        image='busybox:1.36',
        command=['sh', '-c'],
        args=['echo ready > /data/marker'],
        env={'MARKER_TEXT': 'ready'},
        volume_mounts={'shared-data': '/data'},
    )
