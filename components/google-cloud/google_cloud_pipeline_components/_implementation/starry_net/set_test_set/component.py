# Copyright 2024 The Kubeflow Authors. All Rights Reserved.
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
"""Starry Net Set Test Set Component."""

from typing import NamedTuple

from kfp import dsl


@dsl.component(packages_to_install=['tensorflow==2.11.0'])
def set_test_set(
    dataprep_dir: dsl.InputPath(),
) -> NamedTuple('TestSetArtifact', uri=str, artifact=dsl.Artifact):
  # fmt: off
  """Creates test set artifact.

  Args:
    dataprep_dir: The bucket where dataprep artifacts are stored.

  Returns:
    The test set dsl.Artifact.
  """
  import os  # pylint: disable=g-import-not-at-top
  import json  # pylint: disable=g-import-not-at-top
  import tensorflow as tf  # pylint: disable=g-import-not-at-top

  with tf.io.gfile.GFile(
      os.path.join(dataprep_dir, 'big_query_test_set.json')
  ) as f:
    metadata = json.load(f)
  project = metadata['projectId']
  dataset = metadata['datasetId']
  table = metadata['tableId']
  output = NamedTuple('TestSetArtifact', uri=str, artifact=dsl.Artifact)
  uri = f'bq://{project}.{dataset}.{table}'
  artifact = dsl.Artifact(uri=uri, metadata=metadata)
  return output(uri, artifact)  # pylint: disable=too-many-function-args
