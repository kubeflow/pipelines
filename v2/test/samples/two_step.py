# Copyright 2021 Google LLC
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
"""Two step v2-compatible pipeline."""

import kfp
from kfp import compiler, components, dsl
from kfp.components import InputPath, OutputPath


def preprocess(
    uri: str, some_int: int, output_parameter_one: OutputPath(int),
    output_dataset_one: OutputPath('Dataset')
):
    '''Dummy Preprocess Step.'''
    with open(output_dataset_one, 'w') as f:
        f.write('Output dataset')
    with open(output_parameter_one, 'w') as f:
        f.write("{}".format(1234))


def train(
    dataset: InputPath('Dataset'),
    model: OutputPath('Model'),
    num_steps: int = 100
):
    '''Dummy Training Step.'''

    with open(dataset, 'r') as input_file:
        input_string = input_file.read()
        with open(model, 'w') as output_file:
            for i in range(num_steps):
                output_file.write(
                    "Step {}\n{}\n=====\n".format(i, input_string)
                )


preprocess_op = components.create_component_from_func(
    preprocess, base_image='python:3.9'
)
train_op = components.create_component_from_func(train)


@dsl.pipeline(
    pipeline_root='gs://output-directory/v2-artifacts',
    name='two-step-pipeline'
)
def v2_compatible_two_step_pipeline():
    preprocess_task = preprocess_op(uri='uri-to-import', some_int=12)
    train_task = train_op(
        num_steps=preprocess_task.outputs['output_parameter_one'],
        dataset=preprocess_task.outputs['output_dataset_one']
    )


def main(
    pipeline_root: str,
    host: str = 'http://ml-pipeline:8888',
    launcher_image: 'URI' = None
):
    client = kfp.Client(host=host)
    create_run_response = client.create_run_from_pipeline_func(
        v2_compatible_two_step_pipeline,
        mode=dsl.PipelineExecutionMode.V2_COMPATIBLE,
        arguments={kfp.dsl.ROOT_PARAMETER_NAME: pipeline_root},
        launcher_image=launcher_image
    )
    run_response = client.wait_for_run_completion(
        run_id=create_run_response.run_id, timeout=60 * 10
    )
    run = run_response.run
    print('run_id')
    print(run.id)
    print(f"{host}/#/runs/details/{run.id}")
    from pprint import pprint
    pprint(run_response.run)


if __name__ == '__main__':
    import fire
    fire.Fire(main)
