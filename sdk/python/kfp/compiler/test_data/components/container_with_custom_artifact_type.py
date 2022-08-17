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
from kfp.dsl import Input
from kfp.dsl import Output


class CustomArtifact(dsl.Artifact):
    TYPE_NAME = 'platform.CustomArtifact'
    SCHEMA_VERSION = '0.0.0'
    path = 'path'
    uri = 'uri'
    metadata = {}


@dsl.container_component
def container_with_custom_artifact_type(input_: Input[CustomArtifact],
                                        output_: Output[CustomArtifact]):
    return dsl.ContainerSpec(image='python:3.7', command=['echo hello world'])


@dsl.pipeline()
def my_pipeline():
    importer1 = dsl.importer(
        artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
        artifact_class=CustomArtifact,
        reimport=False)
    container_with_custom_artifact_type(input_=importer1.output)


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=container_with_custom_artifact_type,
        package_path=__file__.replace('.py', '.yaml'))

if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from kfp import client
    from kfp import compiler

    warnings.filterwarnings('ignore')
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(pipeline_func=my_pipeline, package_path=ir_file)
    pipeline_name = __file__.split('/')[-1].replace('_', '-').replace('.py', '')
    display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    job_id = f'{pipeline_name}-{display_name}'
    endpoint = 'https://3ac75517948b2661-dot-us-central1.pipelines.googleusercontent.com'
    kfp_client = client.Client(host=endpoint)
    run = kfp_client.create_run_from_pipeline_package(ir_file)
    url = f'{endpoint}/#/runs/details/{run.run_id}'
    print(url)
    webbrowser.open_new_tab(url)
