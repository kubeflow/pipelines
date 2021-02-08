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
"""Tests for kfp.containers.entrypoint_utils module."""
from google.protobuf import json_format
import mock
import os
import unittest

from kfp.dsl import artifact
from kfp import components
from kfp.containers import entrypoint_utils
from kfp.dsl import ontology_artifacts
from kfp.pipeline_spec import pipeline_spec_pb2


def _get_text_from_testdata(filename: str) -> str:
  """Reads the content of a file under testdata."""
  with open(
      os.path.join(os.path.dirname(__file__), 'testdata', filename), 'r') as f:
    return f.read()


def _test_function(
    a: components.InputArtifact('Dataset'),
    b: components.OutputArtifact('Model'),
    c: components.OutputArtifact('Artifact')
):
  """Function used to test signature parsing."""
  pass


_OUTPUT_URIS = {
    'b': 'gs://root/execution/b',
    'c': 'gs://root/execution/c'
}


class EntrypointUtilsTest(unittest.TestCase):

  @mock.patch('kfp.containers._gcs_helper.GCSHelper.read_from_gcs_path')
  def testGetParameterFromOutput(self, mock_read):
    mock_read.return_value = _get_text_from_testdata('executor_output.json')

    self.assertEqual(entrypoint_utils.get_parameter_from_output(
        file_path=os.path.join('testdata', 'executor_output.json'),
        param_name='int_output'
    ), 42)
    self.assertEqual(entrypoint_utils.get_parameter_from_output(
        file_path=os.path.join('testdata', 'executor_output.json'),
        param_name='string_output'
    ), 'hello world!')
    self.assertEqual(entrypoint_utils.get_parameter_from_output(
        file_path=os.path.join('testdata', 'executor_output.json'),
        param_name='float_output'
    ), 12.12)

  @mock.patch('kfp.containers._gcs_helper.GCSHelper.read_from_gcs_path')
  def testGetArtifactFromOutput(self, mock_read):
    mock_read.return_value = _get_text_from_testdata('executor_output.json')

    art = entrypoint_utils.get_artifact_from_output(
        file_path=os.path.join('testdata', 'executor_output.json'),
        output_name='output'
    )
    self.assertIsInstance(art, ontology_artifacts.Model)
    self.assertEqual(art.uri, 'gs://root/execution/output')
    self.assertEqual(art.name, 'test-artifact')
    self.assertEqual(art.get_string_custom_property('test_property'),
                     'test value')

  def testGetOutputArtifacts(self):
    outputs = entrypoint_utils.get_output_artifacts(
        _test_function, _OUTPUT_URIS)
    self.assertSetEqual(set(outputs.keys()), {'b', 'c'})
    self.assertIsInstance(outputs['b'], ontology_artifacts.Model)
    self.assertIsInstance(outputs['c'], artifact.Artifact)
    self.assertEqual(outputs['b'].uri, 'gs://root/execution/b')
    self.assertEqual(outputs['c'].uri, 'gs://root/execution/c')

  def testGetExecutorOutput(self):
    model = ontology_artifacts.Model()
    model.name = 'test-artifact'
    model.uri = 'gs://root/execution/output'
    model.set_string_custom_property('test_property', 'test value')

    executor_output = entrypoint_utils.get_executor_output(
        output_artifacts={'output': model},
        output_params={
            'int_output': 42,
            'string_output': 'hello world!',
            'float_output': 12.12
        })

    # Renormalize the JSON proto read from testdata. Otherwise there'll be
    # mismatch in the way treating int value.
    expected_output = pipeline_spec_pb2.ExecutorOutput()
    expected_output = json_format.Parse(
        text=_get_text_from_testdata('executor_output.json'),
        message=expected_output)

    self.assertDictEqual(
      json_format.MessageToDict(expected_output),
      json_format.MessageToDict(executor_output))

  def testImportFuncFromSource(self):
    fn = entrypoint_utils.import_func_from_source(
        source_path=os.path.join(
            os.path.dirname(__file__), 'testdata', 'pipeline_source.py'),
        fn_name='test_func'
    )
    self.assertEqual(fn(1, 2), 3)

    with self.assertRaisesRegexp(ImportError, '\D+ in \D+ not found in '):
      _ = entrypoint_utils.import_func_from_source(
          source_path=os.path.join('testdata', 'pipeline_source.py'),
          fn_name='non_existing_fn'
      )
