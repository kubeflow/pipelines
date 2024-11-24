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

from kfp.samples.test.utils import KfpTask
from kfp.samples.test.utils import run_pipeline_func
from kfp.samples.test.utils import TestCase
import kfp_server_api
from ml_metadata.proto import Execution

from .component_with_optional_inputs import pipeline


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    component_op_dict = tasks['component-op'].get_dict()

    t.assertEqual(
        {
            'name': 'component-op',
            'inputs': {
                'parameters': {
                    'input_str1': 'Hello',
                    'input_str2': 'World',
                },
            },
            'outputs': {},
            'type': 'system.ContainerExecution',
            'state': Execution.State.COMPLETE,
        }, component_op_dict)


if __name__ == '__main__':
    run_pipeline_func([
        TestCase(
            pipeline_func=pipeline,
            verify_func=verify,
        ),
    ])
