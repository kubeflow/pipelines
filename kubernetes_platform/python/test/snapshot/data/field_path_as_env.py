# Copyright 2023 The Kubeflow Authors
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
    kubernetes.use_field_path_as_env(
        task,
        env_name='KFP_RUN_NAME',
        field_path="metadata.annotations['pipelines.kubeflow.org/run_name']"
    )


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(my_pipeline, __file__.replace('.py', '.yaml'))
