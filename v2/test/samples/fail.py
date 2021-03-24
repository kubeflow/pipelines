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
"""Fail pipeline."""

import kfp
from kfp import compiler, components, dsl
from kfp.components import InputPath, OutputPath


def fail():
    '''Fails'''
    import sys
    sys.exit(1)


fail_op = components.create_component_from_func(
    fail, base_image='alpine:latest'
)


@dsl.pipeline(
    pipeline_root='gs://output-directory/v2-artifacts', name='fail-pipeline'
)
def fail_pipeline():
    preprocess_task = fail_op()


def main(
    pipeline_root: str,
    host: str = 'http://ml-pipeline:8888',
    launcher_image: 'URI' = None
):
    client = kfp.Client(host=host)
    create_run_response = client.create_run_from_pipeline_func(
        fail_pipeline,
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
    assert run.status == 'Failed'
    # TODO: add more MLMD verification


if __name__ == '__main__':
    import fire
    fire.Fire(main)
