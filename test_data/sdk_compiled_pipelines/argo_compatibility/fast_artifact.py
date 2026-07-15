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

from kfp import compiler
from kfp import dsl


@dsl.container_component
def write_artifact(output: dsl.Output[dsl.Dataset]):
    return dsl.ContainerSpec(
        image='alpine:3.23',
        command=['sh', '-c'],
        args=[
            'mkdir -p "$(dirname "$0")"; '
            'printf foo > "$0"; '
            'printf "input:  foo\\n"',
            output.path,
        ],
    )


@dsl.pipeline(name='argo-compatibility-artifact')
def argo_compatibility_artifact():
    write_artifact().set_caching_options(False)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=argo_compatibility_artifact,
        package_path=__file__.replace('.py', '.yaml'),
    )
