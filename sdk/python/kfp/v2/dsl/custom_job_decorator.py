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
"""Custom job decorator.."""

from typing import Callable

from kfp import components


def ai_platform_custom_job(func: Callable):
  """Decorator for Google cloud AI platform custom job component.

  Example usage::

    from kfp.v2 import dsl
    @dsl.ai_platform_custom_job
    def custom_job_op(...):
      return {
        'workerPoolSpecs': [...]
      }

    @dsl.pipeline(name='my-pipeline')
    def pipeline():
      custom_job_op(...)
  """
  return components.create_component_from_func_v2(func=func, is_custom_job=True)
