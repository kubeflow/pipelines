from datetime import datetime
import json
from time import struct_time
from common.common_inputs import COMMON_INPUTS, SageMakerComponentBaseOutputs, SageMakerComponentCommonInputs
from common.sagemaker_component import ComponentMetadata, SageMakerComponent
from tests.unit_tests.tests.common.dummy_spec import DummyInputs, DummyOutputs, DummySpec, ExtraSpec
from tests.unit_tests.tests.common.dummy_component import DummyComponent
import unittest
import os

from typing import Type

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError
from common.sagemaker_component_spec import SageMakerComponentSpec

from common.component_compiler import (
    ArgumentValueSpec,
    ComponentSpec,
    ContainerSpec,
    IOArgs,
    ImplementationSpec,
    InputSpec,
    OutputSpec,
    SageMakerComponentCompiler,
)

class SageMakerComponentMetadataTestCase(unittest.TestCase):
    def test_applies_constants(self):
        self.assertNotEqual(DummyComponent.COMPONENT_NAME, "test1")
        self.assertNotEqual(DummyComponent.COMPONENT_DESCRIPTION, "test2")
        self.assertNotEqual(DummyComponent.COMPONENT_SPEC, str)

        # Run decorator in function form
        ComponentMetadata("test1", "test2", str)(DummyComponent)

        self.assertEqual(DummyComponent.COMPONENT_NAME, "test1")
        self.assertEqual(DummyComponent.COMPONENT_DESCRIPTION, "test2")
        self.assertEqual(DummyComponent.COMPONENT_SPEC, str)

class SageMakerComponentTestCase(unittest.TestCase):
    MOCK_LICENSE_FILE = """Amazon SageMaker Components for Kubeflow Pipelines; version 1.2.3"""
    MOCK_BAD_VERSION_LICENSE_FILE = """Amazon SageMaker Components for Kubeflow Pipelines; version WHATTHE"""
    MOCK_BAD_FORMAT_LICENSE_FILE = """This isn't the first line"""

    @classmethod
    def setUp(cls):
        cls.component = SageMakerComponent()
        cls.boto3_manager_patch = patch("common.sagemaker_component.Boto3Manager")
        cls.boto3_manager_patch.start()

    @classmethod
    def tearDown(cls):
        cls.boto3_manager_patch.stop()

    def test_do_exits_with_error(self):
        self.component._do = MagicMock(side_effect=Exception("Fire!"))

        # Expect the exception is raised up to root
        with self.assertRaises(Exception):
            self.component.Do(COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS)

    def test_do_returns_false_exists(self):
        self.component._do = MagicMock(return_value=False)

        with patch("common.sagemaker_component.sys") as mock_sys:
            self.component.Do(COMMON_INPUTS, DummySpec.OUTPUTS, DummySpec.OUTPUTS)

        mock_sys.exit.assert_called_once_with(1)

    @patch("common.sagemaker_component.strftime", MagicMock(return_value="20201231010203"))
    @patch("common.sagemaker_component.random.choice", MagicMock(return_value="A"))
    def test_generate_unique_timestamped_id(self):
        self.assertEqual("20201231010203-AAAA", self.component._generate_unique_timestamped_id())
        self.assertEqual("20201231010203-AAAAAA", self.component._generate_unique_timestamped_id(size=6))
        self.assertEqual("PREFIX-20201231010203-AAAA", self.component._generate_unique_timestamped_id(prefix="PREFIX"))
        self.assertEqual("PREFIX-20201231010203-AAAAAA", self.component._generate_unique_timestamped_id(prefix="PREFIX", size=6))

    def test_write_all_outputs(self):
        self.component._write_output = MagicMock()

        mock_paths = DummyOutputs(output1="/tmp/output1", output2="/tmp/output2")
        mock_values = DummyOutputs(output1="value1", output2=["value2"])

        self.component._write_all_outputs(mock_paths, mock_values)

        self.component._write_output.assert_has_calls([
            call("/tmp/output1", "value1", json_encode=False),
            call("/tmp/output2", ["value2"], json_encode=True)
        ])

    def test_write_output(self):
        mock_output_path = "/tmp/output1"

        with patch("common.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, "value1")

            mock_path.assert_has_calls([call(mock_output_path), call().parent.mkdir(parents=True, exist_ok=True), call(mock_output_path), call().write_text("value1")])

        with patch("common.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, "value2", json_encode=True)

            mock_path().write_text.assert_any_call("\"value2\"")

        with patch("common.sagemaker_component.Path", MagicMock()) as mock_path:
            self.component._write_output(mock_output_path, ["value1"], json_encode=True)

            mock_path().write_text.assert_any_call(json.dumps(["value1"]))

    @patch("builtins.open")
    def test_get_component_version(self, mock_open):
        mock_open().__enter__().readline.return_value = SageMakerComponentTestCase.MOCK_LICENSE_FILE
        self.assertEqual("1.2.3", self.component._get_component_version())

        mock_open().__enter__().readline.return_value = SageMakerComponentTestCase.MOCK_BAD_FORMAT_LICENSE_FILE
        self.assertEqual("NULL", self.component._get_component_version())

        mock_open().__enter__().readline.return_value = SageMakerComponentTestCase.MOCK_BAD_VERSION_LICENSE_FILE
        self.assertEqual("NULL", self.component._get_component_version())

    @patch("common.sagemaker_component.yaml")
    @patch("builtins.open")
    def test_get_request_template(self, mock_open, mock_yaml):
        mock_yaml_blob = "keyname: value"

        mock_open().__enter__.return_value = mock_yaml_blob
        mock_yaml.safe_load.return_value = {"keyname": "value"}

        with patch("common.sagemaker_component.SageMakerComponent._get_common_path", MagicMock(return_value="/my/path")):
            response = self.component._get_request_template("my-template")

        mock_open.assert_any_call("/my/path/templates/my-template.template.yaml", "r")
        mock_yaml.safe_load.assert_called_once_with(mock_yaml_blob)
        self.assertEqual(response, {"keyname": "value"})