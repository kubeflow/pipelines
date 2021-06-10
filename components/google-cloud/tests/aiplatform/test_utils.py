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

from inspect import signature
import json
from typing import Optional, Dict, Tuple, Union, List, ForwardRef
import unittest
from google.cloud import aiplatform
from google_cloud_pipeline_components.aiplatform import utils

INIT_KEY = 'init'
METHOD_KEY = 'method'


class UtilsTests(unittest.TestCase):

    def setUp(self):
        super(UtilsTests, self).setUp()

    def _test_method(
        self, credentials: Optional, sync: bool, input_param: str,
        input_name: str
    ):
        """Test short description.

            Long descirption

            Args:
                credentials: credentials
                sync: sync
                input_param: input_param
                input_name:input_name
            """

    def test_get_forward_reference_with_annotation_str(self):
        annotation = aiplatform.Model.__name__

        results = utils.get_forward_reference(annotation)
        self.assertEqual(results, aiplatform.Model)

    def test_get_forward_reference_with_annotation_forward_reference(self):
        annotation = ForwardRef(aiplatform.Model.__name__)

        results = utils.get_forward_reference(annotation)
        self.assertEqual(results, aiplatform.Model)

    def test_resolve_annotation_with_annotation_class(self):
        annotation = aiplatform.Model

        results = utils.resolve_annotation(annotation)
        self.assertEqual(results, annotation)

    def test_resolve_annotation_with_annotation_foward_str_reference(self):
        annotation = aiplatform.Model.__name__

        results = utils.resolve_annotation(annotation)
        self.assertEqual(results, aiplatform.Model)

    def test_resolve_annotation_with_annotation_foward_typed_reference(self):
        annotation = ForwardRef(aiplatform.Model.__name__)

        results = utils.resolve_annotation(annotation)
        self.assertEqual(results, aiplatform.Model)

    def test_resolve_annotation_with_annotation_type_union(self):
        annotation = Union[Dict, None]

        results = utils.resolve_annotation(annotation)
        self.assertEqual(results, Dict)

    def test_resolve_annotation_with_annotation_type_empty(self):
        annotation = None

        results = utils.resolve_annotation(annotation)
        self.assertEqual(results, None)

    def test_is_serializable_to_json_with_serializable_type(self):
        annotation = Dict

        results = utils.is_serializable_to_json(annotation)
        self.assertTrue(results)

    def test_is_serializable_to_json_with_not_serializable_type(self):
        annotation = Tuple

        results = utils.is_serializable_to_json(annotation)
        self.assertFalse(results)

    def test_is_mb_sdk_resource_noun_type_with_not_noun_type(self):
        annotation = Tuple

        results = utils.is_serializable_to_json(annotation)
        self.assertFalse(results)

    def test_is_mb_sdk_resource_noun_type_with_resource_noun_type(self):
        mb_sdk_type = aiplatform.Model

        results = utils.is_mb_sdk_resource_noun_type(mb_sdk_type)
        self.assertTrue(results)

    def test_get_serializer_with_serializable_type(self):
        annotation = Dict

        results = utils.get_serializer(annotation)
        self.assertEqual(results, json.dumps)

    def test_get_serializer_with_not_serializable_type(self):
        annotation = Tuple

        results = utils.get_serializer(annotation)
        self.assertEqual(results, None)

    def test_get_deserializer_with_serializable_type(self):
        annotation = Dict

        results = utils.get_deserializer(annotation)
        self.assertEqual(results, json.loads)

    def test_get_deserializer_with_not_serializable_type(self):
        annotation = Tuple

        results = utils.get_deserializer(annotation)
        self.assertEqual(results, None)

    def test_map_resource_to_metadata_type_with_mb_sdk_type(self):
        mb_sdk_type = aiplatform.Model

        parameter_name, parameter_type = utils.map_resource_to_metadata_type(
            mb_sdk_type
        )
        self.assertEqual(parameter_name, 'model')
        self.assertEqual(parameter_type, 'Model')

    def test_map_resource_to_metadata_type_with_serializable_type(self):
        mb_sdk_type = List

        parameter_name, parameter_type = utils.map_resource_to_metadata_type(
            mb_sdk_type
        )
        self.assertEqual(parameter_name, 'exported_dataset')
        self.assertEqual(parameter_type, 'Dataset')

    def test_map_resource_to_metadata_type_with__Dataset_type(self):
        mb_sdk_type = '_Dataset'

        parameter_name, parameter_type = utils.map_resource_to_metadata_type(
            mb_sdk_type
        )
        self.assertEqual(parameter_name, 'dataset')
        self.assertEqual(parameter_type, 'Dataset')

    def test_is_resource_name_parameter_name_with_display_name(self):
        param_name = 'display_name'
        self.assertFalse(utils.is_resource_name_parameter_name(param_name))

    def test_is_resource_name_parameter_name_with_encryption_spec_key_name(
        self
    ):
        param_name = 'display_name'
        self.assertFalse(utils.is_resource_name_parameter_name(param_name))

    def test_is_resource_name_parameter_name_with_resource_name(self):
        param_name = 'testresource_name'
        self.assertTrue(utils.is_resource_name_parameter_name(param_name))

    def test_filter_signature_with_PARAMS_TO_REMOVE(self):

        def test_method(self, credentials: Optional, sync, init_param: str):
            pass

        filtered_signature = utils.filter_signature(signature(test_method))
        expected_output_str = "<Signature (init_param: str)>"
        self.assertEqual(repr(filtered_signature), expected_output_str)

    def test_filter_signature_with_resouce_name(self):

        def test_method(model_name: str):
            pass

        param_map = {}
        filtered_signature = utils.filter_signature(
            signature=signature(test_method),
            is_init_signature=True,
            self_type=str,
            component_param_name_to_mb_sdk_param_name=param_map
        )
        expected_output_str = "<Signature (model: str)>"
        self.assertEqual(repr(filtered_signature), expected_output_str)

    def test_signatures_union_with_(self):

        def test_init(init_param: str):
            pass

        def test_method(method_param: str):
            pass

        init_signature = signature(test_init)
        method_signature = signature(test_method)
        results = utils.signatures_union(init_signature, method_signature)
        expected_results = "<Signature (init_param: str, method_param: str)>"
        self.assertEqual(repr(results), expected_results)

    def test_signatures_union_with_PARAMS_TO_REMOVE(self):

        test_docstring_method_signature = signature(self._test_method)
        docstring = self._test_method.__doc__
        results = utils.filter_docstring_args(
            signature=test_docstring_method_signature,
            docstring=docstring,
            is_init_signature=True
        )
        expected_results = {'input': 'input_name', 'input_param': 'input_param'}
        self.assertDictEqual(results, expected_results)

    def test_generate_docstring_with_PARAMS_TO_REMOVE(self):
        args_dict = {'input': 'input_name', 'input_param': 'input_param'}
        param_map = {}
        converted_method_signature = utils.filter_signature(
            signature(self._test_method),
            is_init_signature=True,
            component_param_name_to_mb_sdk_param_name=param_map
        )

        results = utils.generate_docstring(
            args_dict=args_dict,
            signature=converted_method_signature,
            method_docstring=self._test_method.__doc__
        )
        expected_results = "".join(
            "Test short description.\n"
            "Long descirption\n\nArgs:\n"
            "    input:\n"
            "        input_name\n"
            "    input_param:\n"
            "        input_param\n"
        )
        self.assertEqual(results, expected_results)
