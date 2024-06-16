# Copyright 2024 The Kubeflow Authors
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
from kfp import kubernetes


@dsl.component
def comp():
    pass

@dsl.pipeline
def my_pipeline():
    task = comp()
    kubernetes.empty_dir_mount(
            task, 
            volume_name='emptydir-vol-1', 
            mount_path='/mnt/my_vol_1',
            medium='Memory',
            size_limit='1Gi'
        )

if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(my_pipeline, __file__.replace('.py', '.yaml'))
