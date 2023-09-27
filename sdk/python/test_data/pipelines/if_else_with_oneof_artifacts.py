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

if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform
    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(
        pipeline_func=outer_pipeline, package_path=ir_file)
    pipeline_name = __file__.split('/')[-1].replace('_', '-').replace('.py', '')
    display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    job_id = f'{pipeline_name}-{display_name}'
    aiplatform.PipelineJob(
        template_path=ir_file,
        pipeline_root='gs://cjmccarthy-kfp-default-bucket',
        display_name=pipeline_name,
        job_id=job_id).submit()
    url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=271009669852'
    webbrowser.open_new_tab(url)
