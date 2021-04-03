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

from .two_step import two_step_pipeline
from .util import run_pipeline_func


def verify(run, run_id: str):
    assert run.status == 'Succeeded'
    # TODO(Bobgy): verify MLMD status


run_pipeline_func(pipeline_func=two_step_pipeline, verify_func=verify)
