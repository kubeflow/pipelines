# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Executor for running Vertex Notification Email components."""


def main():
  """Raises an error when notifications are requested on Kubeflow Pipelines.

  The notification email component works only on Vertex Pipelines. This function
  raises an exception when this component is used on Kubeflow Pipelines.
  """
  raise NotImplementedError('The notification email component is supported '
                            'only on Vertex Pipelines.')
