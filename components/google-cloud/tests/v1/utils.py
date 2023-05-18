# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
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
"""Utilities for testing."""
import copy
import json
from typing import Any, Optional, Set

from google_cloud_pipeline_components import _image
from kfp import components
import yaml

from google.protobuf import json_format
import unittest


def assert_pipeline_equals_golden(
    test_case: unittest.TestCase,
    compilable,
    comparison_file: str,
) -> None:
  """Compare a compilable pipeline/component against a golden snapshot comparison file containing a known valid PipelineSpec.

  Permits the compiled output to have a schema that has migrated past the
  comparison file. Skips comparison of fields that do not need to be compared to
  assert equality.

  Args:
    test_case: An instance of a TestCase.
    compilable: Pipeline/component.
    comparison_file: Path to a JSON golden snapshot of a the compiled
      PipelineSpec.
  """

  expected_pipeline_spec_dict = json_format.MessageToDict(
      components.load_component_from_file(comparison_file).pipeline_spec
  )
  actual_pipeline_spec_dict = json_format.MessageToDict(
      compilable.pipeline_spec
  )
  expected_pipeline_spec_dict['sdkVersion'] = 'bye'
  actual_pipeline_spec_dict['sdkVersion'] = 'hi'
  ignore_fields = {'sdkVersion'}
  compare_pipeline_spec_dicts(
      test_case,
      actual_pipeline_spec_dict,
      expected_pipeline_spec_dict,
      comparison_file,
      ignore_fields,
  )


def _actual_and_expected_images_have_same_name(
    actual: Any, expected: Any
) -> bool:
  return (
      isinstance(actual, str)
      and isinstance(expected, str)
      and actual.startswith(_image.GCPC_IMAGE_NAME)
      and expected.startswith(_image.GCPC_IMAGE_NAME)
  )


def compare_pipeline_spec_dicts(
    test_case: unittest.TestCase,
    actual: dict,
    expected: dict,
    comparison_file: Optional[str] = None,
    ignore_fields: Optional[Set[str]] = None,
) -> None:
  """Compares two PipelineSpec dictionaries.

  Permits actual to have a proto schema evolution that
  is ahead of expected. If KFP SDK adds a field to PipelineSpec, but the golden
  snapshot hasn't been updated, the test case will not fail. Prints out the
  actual JSON for easy copy and paste into the golden snapshot file. Note that
  actual and expected are treated differently and are not
  interchangeable.

  Args:
    test_case: An instance of a TestCase.
    actual: Actual PipelineSpec as a dict.
    expected: Expected PipelineSpec as a dict.
    comparison_file: Path to a JSON golden snapshot of a the compiled
      PipelineSpec.
    ignore_fields: If a field's key is in ignore_fields it will not be used to
      assert equality.
  """
  is_json = comparison_file.endswith('.json')
  original_actual = copy.deepcopy(actual)
  ignore_fields = ignore_fields or None

  def make_copypaste_message(
      actual_pipeline_spec_json: dict,
      is_json: bool = True,
  ) -> str:
    if is_json:
      proposed_file_contents = json.dumps(
          actual_pipeline_spec_json,
      )
    else:
      proposed_file_contents = yaml.dump(actual_pipeline_spec_json)

    return (
        '\n\nTo update the JSON to the new version, copy and paste the'
        f' following into the golden snapshot file {comparison_file or ""}. Be'
        ' sure the change is what you expect.\n'
        + proposed_file_contents
    )

  def compare_json_dicts(
      test_case: unittest.TestCase,
      actual: Any,
      expected: Any,
  ) -> None:
    if type(actual) is not type(expected):
      print(make_copypaste_message(original_actual, is_json))
    test_case.assertEqual(
        type(actual),
        type(expected),
        f'Types do not match: {type(actual)} != {type(expected)}',
    )
    if isinstance(actual, dict):
      for key in expected:
        if key in ignore_fields:
          continue
        if key not in actual:
          print(make_copypaste_message(original_actual, is_json))
        test_case.assertIn(
            key,
            actual,
            f'Key "{key}" not found in first json object',
        )
        compare_json_dicts(test_case, actual[key], expected[key])
    elif isinstance(actual, list):
      test_case.assertEqual(
          len(actual), len(expected), 'Lists are of different lengths'
      )
      for i in range(len(actual)):
        compare_json_dicts(test_case, actual[i], expected[i])
    else:
      if actual != expected and not _actual_and_expected_images_have_same_name(
          actual, expected
      ):
        print(make_copypaste_message(original_actual, is_json))
        test_case.assertEqual(
            actual,
            expected,
            f'Values do not match: {actual} != {expected}',
        )

  compare_json_dicts(test_case, actual, expected)
