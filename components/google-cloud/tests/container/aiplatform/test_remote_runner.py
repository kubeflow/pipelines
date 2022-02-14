# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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
"""Test remote_runner module."""

import json
import os
import unittest
from unittest import mock
from google.cloud import aiplatform
from google_cloud_pipeline_components.container.utils.execution_context import ExecutionContext
from google_cloud_pipeline_components.container.aiplatform import remote_runner
from google_cloud_pipeline_components.aiplatform import utils

INIT_KEY = 'init'
METHOD_KEY = 'method'

class TestAutoMLImageTrainingJob(aiplatform.AutoMLImageTrainingJob):
    def __init__(self, *_0, **_1):
        return
    def run(self):
        return
    def cancel(self):
        return

class RemoteRunnerTests(unittest.TestCase):

    def setUp(self):
        super(RemoteRunnerTests, self).setUp()

    def test_split_args_separates_init_and_method_args(self):
        test_kwargs = {
            f"{INIT_KEY}.test.class.init_1": 1,
            f"{INIT_KEY}.testclass.init_2": 2,
            f"{METHOD_KEY}.testclass.method_1": 1,
            f"{METHOD_KEY}.testclass.method_2": 2
        }

        init_args, method_args = remote_runner.split_args(test_kwargs)

        self.assertDictEqual(init_args, {'init_1': 1, 'init_2': 2})
        self.assertDictEqual(method_args, {'method_1': 1, 'method_2': 2})

    def test_split_args_on_emtpy_init_returns_empty_dict(self):
        test_kargs = {
            f"{INIT_KEY}.test.class.init_1": 1,
            f"{INIT_KEY}.testclass.init_2": 2,
        }

        init_args, method_args = remote_runner.split_args(test_kargs)

        self.assertDictEqual(init_args, {'init_1': 1, 'init_2': 2})
        self.assertDictEqual(method_args, {})

    def test_split_args_on_emtpy_method_returns_empty_dict(self):
        test_kargs = {
            f"{METHOD_KEY}.testclass.method_1": 1,
            f"{METHOD_KEY}.testclass.method_2": 2
        }

        init_args, method_args = remote_runner.split_args(test_kargs)

        self.assertDictEqual(init_args, {})
        self.assertDictEqual(method_args, {'method_1': 1, 'method_2': 2})

    @mock.patch('builtins.open', new_callable=mock.mock_open())
    @mock.patch.object(os, "makedirs", autospec=True)
    def test_write_to_artifact_preserves_uri_name_metadata(
        self, mock_makedirs, mock_open
    ):
        executor_input_dict = {
            "outputs": {
                "artifacts": {
                    "dataset": {
                        "artifacts": [{
                            "name": "test_name",
                            "type": {
                                "instanceSchema":
                                    "properties:\ntitle: kfp.Dataset\ntype: object\n"
                            },
                            "uri": "gs://test_path/dataset"
                        }]
                    }
                },
                "outputFile": "/gcs/test_path/executor_output.json"
            }
        }
        expected_output = {
            "artifacts": {
                "dataset": {
                    "artifacts": [{
                        "name":
                            "test_name",
                        "uri":
                            "https://us-central1-aiplatform.googleapis.com/v1/projects/513263813639/locations/us-central1/models/7027708888837259264",
                        "metadata": {
                            "resourceName": "projects/513263813639/locations/us-central1/models/7027708888837259264"
                        }
                    }]
                }
            }
        }

        remote_runner.write_to_artifact(
            executor_input_dict,
            "projects/513263813639/locations/us-central1/models/7027708888837259264"
        )

        mock_open.return_value.__enter__().write.assert_called_once_with(
            json.dumps(expected_output)
        )

    @mock.patch('builtins.open', new_callable=mock.mock_open())
    @mock.patch.object(os, "makedirs", autospec=True)
    def test_write_to_artifact_with_no_artifacts_writes_emtpy_dict(
        self, mock_makedirs, mock_open
    ):
        remote_runner.write_to_artifact({
            "outputs": {
                "artifacts": {},
                "outputFile": "/gcs/test_path/executor_output.json"
            }
        }, "test_resource_uri")

        mock_open.return_value.__enter__().write.assert_called_once_with(
            json.dumps({})
        )

    def test_resolve_input_args_resource_noun_removes_gcs_prefix(self):
        type_to_resolve = aiplatform.Model
        value = '/gcs/test_resource_name'
        expected_result = 'test_resource_name'

        result = remote_runner.resolve_input_args(value, type_to_resolve)
        self.assertEqual(result, expected_result)

    def test_resolve_input_args_resource_noun_not_changed(self):
        type_to_resolve = aiplatform.base.VertexAiResourceNoun
        value = 'test_resource_name'
        expected_result = 'test_resource_name'

        result = remote_runner.resolve_input_args(value, type_to_resolve)
        self.assertEqual(result, expected_result)

    def test_resolve_input_args_not_type_to_resolve_not_changed(self):
        type_to_resolve = object
        value = '/gcs/test_resource_name'
        expected_result = '/gcs/test_resource_name'

        result = remote_runner.resolve_input_args(value, type_to_resolve)
        self.assertEqual(result, expected_result)

    def test_resolve_init_args_key_does_not_end_with_name_not_changed(self):
        key = 'resource'
        value = '/gcs/test_resource_name'
        expected_result = '/gcs/test_resource_name'

        result = remote_runner.resolve_init_args(key, value)
        self.assertEqual(result, expected_result)

    def test_resolve_init_args_key_ends_with_name_removes_gcs_prefix(self):
        key = 'resource_name'
        value = '/gcs/test_resource_name'
        expected_result = 'test_resource_name'

        result = remote_runner.resolve_init_args(key, value)
        self.assertEqual(result, expected_result)

    def test_make_output_with_not_resource_name_returns_serialized_list_json(
        self
    ):
        output_object = ["a", 2]
        expected_result = '["a", 2]'
        result = remote_runner.make_output(output_object)
        self.assertEqual(result, expected_result)

    # TODO(SinaChavoshi): Reenable this when removing the auth dependency
    # def test_make_output_with_resource_name_returns_resource_name_value(self):
    #     output_object = aiplatform.Model._empty_constructor(
    #         project='test-project'
    #     )
    #     output_object._gca_resource = aiplatform.gapic.Model(
    #         name='test_resource_name'
    #     )
    #     expected_result = 'test_resource_name'
    #     result = remote_runner.make_output(output_object)
    #     self.assertEqual(result, expected_result)

    def test_cast_with_str_value_to_bool(self):
        expected_result = True
        result = remote_runner.cast('True', bool)
        self.assertEqual(result, expected_result)

    def test_cast_with_str_value_to_list(self):
        expected_result = int(2)
        result = remote_runner.cast('2', int)
        self.assertIsInstance(result, type(expected_result))
        self.assertEqual(result, expected_result)

    @mock.patch.object(
        utils, "resolve_annotation", autospec=True, return_value=bool
    )
    @mock.patch.object(
        utils, "get_deserializer", autospec=True, return_value=bool
    )
    @mock.patch.object(
        remote_runner, "resolve_input_args", autospec=True, return_value="True"
    )
    @mock.patch.object(
        remote_runner, "resolve_init_args", autospec=True, return_value="True"
    )
    def test_prepare_parameters_with_init_params(
        self, mock_resolve_init_args, mock_resolve_input_args,
        mock_get_deserializer, mock_resolve_annotation
    ):

        def sample_method(input: bool):
            pass

        input_kwargs = {"input": "True"}
        expected_kwargs = {"input": True}
        remote_runner.prepare_parameters(input_kwargs, sample_method, True)
        mock_resolve_annotation.assert_called_once_with(bool)
        mock_get_deserializer.assert_called_once_with(bool)
        mock_resolve_init_args.assert_called_once_with("input", "True")
        mock_resolve_input_args.assert_not_called()
        self.assertDictEqual(input_kwargs, expected_kwargs)

    @mock.patch.object(
        utils, "resolve_annotation", autospec=True, return_value=bool
    )
    @mock.patch.object(
        utils, "get_deserializer", autospec=True, return_value=None
    )
    @mock.patch.object(
        remote_runner, "resolve_init_args", autospec=True, return_value="True"
    )
    def test_prepare_parameters_with_init_params_no_serializer(
        self, mock_resolve_init_args, mock_get_deserializer,
        mock_resolve_annotation
    ):

        def sample_method(input: bool):
            pass

        input_kwargs = {"input": "True"}
        expected_kwargs = {"input": True}
        remote_runner.prepare_parameters(input_kwargs, sample_method, True)
        mock_resolve_annotation.assert_called_once_with(bool)
        mock_get_deserializer.assert_called_once_with(bool)
        mock_resolve_init_args.assert_called_once_with("input", "True")
        self.assertDictEqual(input_kwargs, expected_kwargs)

    @mock.patch.object(
        utils, "resolve_annotation", autospec=True, return_value=bool
    )
    @mock.patch.object(
        utils, "get_deserializer", autospec=True, return_value=bool
    )
    @mock.patch.object(
        remote_runner, "resolve_input_args", autospec=True, return_value="True"
    )
    @mock.patch.object(
        remote_runner, "resolve_init_args", autospec=True, return_value="True"
    )
    def test_prepare_parameters_with_method_params(
        self, mock_resolve_init_args, mock_resolve_input_args,
        mock_get_deserializer, mock_resolve_annotation
    ):

        def sample_method(input: bool):
            pass

        input_kwargs = {"input": "True"}
        expected_kwargs = {"input": True}
        remote_runner.prepare_parameters(input_kwargs, sample_method, False)
        mock_resolve_annotation.assert_called_once_with(bool)
        mock_get_deserializer.assert_called_once_with(bool)
        mock_resolve_input_args.assert_called_once_with('True', bool)
        mock_resolve_init_args.assert_not_called()
        self.assertDictEqual(input_kwargs, expected_kwargs)

    @mock.patch.object(
        aiplatform, "AutoMLImageTrainingJob", autospec=TestAutoMLImageTrainingJob
    )
    @mock.patch.object(
        remote_runner, "make_output", autospec=True, return_value="True"
    )
    @mock.patch.object(
        remote_runner, "write_to_artifact", autospec=True, return_value="True"
    )
    @mock.patch.object(
        remote_runner, "prepare_parameters", autospec=True, return_value="True"
    )
    @mock.patch.object(
        remote_runner, "split_args", autospec=True, return_value="True"
    )
    @mock.patch.object(ExecutionContext, '__init__', autospec=True)
    def test_runner_cancel(
        self, mock_execution_context, mock_split_args, mock_prepare_parameters,
        _0, _1, mock_training_job
    ):
        mock_split_args.return_value = ({'project': 'project_id'}, {})
        mock_training_job_instance = mock.Mock()
        mock_training_job.return_value = mock_training_job_instance
        mock_execution_context.return_value = None

        remote_runner.runner("AutoMLImageTrainingJob", "run", None, None)
        self.assertEqual(
            mock_execution_context.call_args[1]['on_cancel'],
            mock_training_job_instance.cancel)
