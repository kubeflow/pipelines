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
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.component
def flip_coin() -> str:
    import random
    return 'heads' if random.randint(0, 1) == 0 else 'tails'


@dsl.component
def param_to_artifact(val: str, a: Output[Artifact]):
    with open(a.path, 'w') as f:
        f.write(val)


@dsl.component
def print_artifact(a: Input[Artifact]):
    with open(a.path) as f:
        print(f.read())


@dsl.pipeline
def flip_coin_pipeline() -> Artifact:
    flip_coin_task = flip_coin()
    with dsl.If(flip_coin_task.output == 'heads'):
        t1 = param_to_artifact(val=flip_coin_task.output)
    with dsl.Else():
        t2 = param_to_artifact(val=flip_coin_task.output)
    oneof = dsl.OneOf(t1.outputs['a'], t2.outputs['a'])
    print_artifact(a=oneof)
    return oneof


@dsl.pipeline
def outer_pipeline():
    flip_coin_task = flip_coin_pipeline()
    print_artifact(a=flip_coin_task.output)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=outer_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
