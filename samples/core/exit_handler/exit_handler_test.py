# Copyright 2021 The Kubeflow Authors
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

# %%

import unittest
from pprint import pprint

import kfp
import kfp_server_api

from .exit_handler import pipeline_exit_handler
from kfp.samples.test.utils import run_pipeline_func, TestCase, KfpMlmdClient


def verify(mlmd_connection_config, run: kfp_server_api.ApiRun, **kwargs):
    t = unittest.TestCase()
    t.maxDiff = None  # we always want to see full diff

    t.assertEqual(run.status, 'Succeeded')

    # Verify MLMD state
    client = KfpMlmdClient(mlmd_connection_config=mlmd_connection_config)
    tasks = client.get_tasks(run_id=run.id)
    task_names = [*tasks.keys()]
    t.assertEqual(task_names, ['echo-msg', 'print-file', 'download-from-gcs'])

    for task in task_names:
        pprint(f'======= {task} =======')
        pprint(tasks.get(task).get_dict())

    t.assertEqual(
        tasks.get('echo-msg').inputs.parameters.get('msg'),
        'exit!',
    )


# %%

if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline_exit_handler,
            mode=kfp.dsl.PipelineExecutionMode.V1_LEGACY,
        ),
    ])
