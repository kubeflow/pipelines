# Copyright 2022 The Kubeflow Authors
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
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Model
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def container_with_artifact_output(
        num_epochs: int,  # built-in types are parsed as inputs
        model: Output[Model],
        model_config_path: OutputPath(str),
):
    return ContainerSpec(
        image='gcr.io/my-image',
        command=['sh', 'run.sh'],
        args=[
            '--epochs',
            num_epochs,
            '--model_path',
            model.uri,
            '--model_metadata',
            model.metadata,
            '--model_config_path',
            model_config_path,
        ])


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=container_with_artifact_output,
        package_path=__file__.replace('.py', '.yaml'))
