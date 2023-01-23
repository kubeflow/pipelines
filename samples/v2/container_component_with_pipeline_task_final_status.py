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
from kfp import dsl


@dsl.component
def maybe_fail():
    import random
    import sys

    sys.exit(random.choice([0, 1]))


@dsl.container_component
def echo_status(status: dsl.PipelineTaskFinalStatus):
    return dsl.ContainerSpec(image='alpine', command=['echo', status])


@dsl.pipeline
def my_pipeline():
    echo_status_task = echo_status()
    with dsl.ExitHandler(echo_status_task):
        maybe_fail_task = maybe_fail()


if __name__ == '__main__':
    from kfp import compiler

    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
