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
"""Utilities for testing local execution."""

import unittest

from absl.testing import parameterized
from google.protobuf import json_format
from kfp import components
from kfp import dsl
from kfp.dsl import base_component
from kfp.local import config


class LocalRunnerEnvironmentTestCase(parameterized.TestCase):

    def tearDown(self):
        config.LocalExecutionConfig.instance = None

    @classmethod
    def setUpClass(cls):
        base_component.TEMPORARILY_BLOCK_LOCAL_EXECUTION = False


def compile_and_load_component(
    base_component: dsl.base_component.BaseComponent,
) -> dsl.yaml_component.YamlComponent:
    """Compiles a component to PipelineSpec and reloads it as a
    YamlComponent."""
    return components.load_component_from_text(
        json_format.MessageToJson(base_component.pipeline_spec))


def assert_artifacts_equal(
    test_class: unittest.TestCase,
    a1: dsl.Artifact,
    a2: dsl.Artifact,
) -> None:
    test_class.assertEqual(a1.name, a2.name)
    test_class.assertEqual(a1.uri, a2.uri)
    test_class.assertEqual(a1.metadata, a2.metadata)
    test_class.assertEqual(a1.schema_title, a2.schema_title)
    test_class.assertEqual(a1.schema_version, a2.schema_version)
    test_class.assertIsInstance(a1, type(a2))
