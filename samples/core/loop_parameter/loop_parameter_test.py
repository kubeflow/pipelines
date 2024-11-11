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

from __future__ import annotations

import unittest

from kfp.samples.test.utils import KfpTask
from kfp.samples.test.utils import run_pipeline_func
from kfp.samples.test.utils import TestCase
import kfp_server_api
from loop_parameter import my_pipeline


def verify(t: unittest.TestCase, run: kfp_server_api.ApiRun,
           tasks: dict[str, KfpTask], **kwargs):
    t.assertEqual(run.status, 'Succeeded')
    # assert DAG structure
    t.assertCountEqual(['generate-op', 'print-op', 'for-loop-1'], tasks.keys())
    # assert all iteration outputs
    t.assertCountEqual(['110', '220', '330', '440'], [
        x.children['print-op-2'].outputs.parameters['Output']
        for x in tasks['for-loop-1'].children.values()
    ])


run_pipeline_func([
    TestCase(pipeline_func=my_pipeline, verify_func=verify),
])
