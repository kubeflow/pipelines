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
"""Component with optinal and default input v2 engine pipeline."""

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

from .component_with_optional_inputs import pipeline


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    component_op_dict = tasks['component_op'].get_dict()
    for artifact in component_op_dict.get('outputs').get('artifacts'):
        # pop metadata here because the artifact which got re-imported may have metadata with uncertain data
        if artifact.get('metadata') is not None:
            artifact.pop('metadata')

    t.assertEqual(
        {
            'name': 'component_op',
            'inputs': {
                'parameters': {
                    'input1': 'Hello',
                    'input2': 'World'
                }
            },
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, component_op_dict)

if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline,
            verify_func=verify,
            mode=kfp.dsl.PipelineExecutionMode.V2_ENGINE,
        ),
    ])