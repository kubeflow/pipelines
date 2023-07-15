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

import unittest

from kfp.dsl import container_component_artifact_channel
from kfp.dsl import placeholders


class TestContainerComponentArtifactChannel(unittest.TestCase):

    def test_correct_placeholder_and_attribute_error(self):
        in_channel = container_component_artifact_channel.ContainerComponentArtifactChannel(
            'input', 'my_dataset')
        out_channel = container_component_artifact_channel.ContainerComponentArtifactChannel(
            'output', 'my_result')
        self.assertEqual(
            in_channel.uri._to_string(),
            placeholders.InputUriPlaceholder('my_dataset')._to_string())
        self.assertEqual(
            out_channel.path._to_string(),
            placeholders.OutputPathPlaceholder('my_result')._to_string())
        self.assertEqual(
            out_channel.metadata._to_string(),
            placeholders.OutputMetadataPlaceholder('my_result')._to_string())
        self.assertRaisesRegex(AttributeError,
                               r'Cannot access artifact attribute "name"',
                               lambda: in_channel.name)
        self.assertRaisesRegex(AttributeError,
                               r'Cannot access artifact attribute "channel"',
                               lambda: out_channel.channel)
