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
"""Tests for kfp.dsl.io_types."""

import unittest
from typing import List, Optional

from kfp.dsl import io_types
from kfp.dsl.io_types import Artifact, Dataset, Input, Output


class IOTypesTest(unittest.TestCase):

  def test_is_artifact_annotation(self):
    # Artifact annotations.
    self.assertTrue(io_types.is_artifact_annotation(Input[Artifact]))
    self.assertTrue(io_types.is_artifact_annotation(Output[Artifact]))
    self.assertTrue(io_types.is_artifact_annotation(Output[Dataset]))
    self.assertTrue(io_types.is_artifact_annotation(Output['MyArtifact']))

    # Non-artifact annotations.
    self.assertFalse(io_types.is_artifact_annotation(float))
    self.assertFalse(io_types.is_artifact_annotation('String'))
    self.assertFalse(io_types.is_artifact_annotation(List[str]))
    self.assertFalse(io_types.is_artifact_annotation(Optional[str]))


if __name__ == '__main__':
  unittest.main()
