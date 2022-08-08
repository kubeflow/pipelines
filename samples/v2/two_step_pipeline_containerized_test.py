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
"""Pipeline container no input v2 engine pipeline."""

from __future__ import annotations

import unittest

import kfp.deprecated as kfp
from kfp.samples.test.utils import KfpTask
from kfp.samples.test.utils import run_pipeline_func
from kfp.samples.test.utils import TaskInputs
from kfp.samples.test.utils import TaskOutputs
from kfp.samples.test.utils import TestCase
import kfp_server_api
from ml_metadata.proto import Execution

from .two_step_pipeline_containerized import two_step_pipeline_containerized


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    component1_dict = tasks['component1'].get_dict()
    component2_dict = tasks['component2'].get_dict()
    for artifact in component1_dict.get('outputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        if artifact.get('metadata') is not None:
            artifact.pop('metadata')
    for artifact in component2_dict.get('inputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        if artifact.get('metadata') is not None:
            artifact.pop('metadata')

    t.assertEqual(
        {
            'name': 'component1',
            'inputs': {
                'parameters': {
                    'text': 'hi'
                }
            },
            'outputs': {
                'artifacts': [{
                    'name': 'output_gcs',
                    'type': 'system.Dataset'
                }],
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, component1_dict)

    t.assertEqual(
        {
            'name': 'component2',
            'inputs': {
                'artifacts': [{
                    'name': 'input_gcs',
                    'type': 'system.Dataset'
                }],
            },
            'outputs': {},
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, component2_dict)


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=two_step_pipeline_containerized,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])
