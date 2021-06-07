from common.spec_input_parsers import SpecInputParsers
from dataclasses import dataclass
from common.common_inputs import (
    SageMakerComponentBaseInputs,
    SageMakerComponentBaseOutputs,
    SageMakerComponentInput,
    SageMakerComponentOutput,
)
from tests.unit_tests.tests.common.dummy_spec import (
    AllInputTypes,
    DummyInputs,
    DummyOutputs,
    DummySpec,
    ExtraSpec,
    NoOutputs,
)
import unittest
import os

from unittest.mock import patch, call, Mock, MagicMock, mock_open, ANY
from botocore.exceptions import ClientError

from common.sagemaker_component_spec import SageMakerComponentSpec


class SageMakerComponentSpecTestCase(unittest.TestCase):
    DUMMY_INPUT_ARGS = ["--input1", "string", "--input2", "1"]

    def test_validates_spec_successfully(self):
        # Will raise an exception if invalid
        DummySpec._validate_spec()

    def test_spec_constructor(self):
        spec = DummySpec(["--input2", "123"])

        self.assertEqual(spec._inputs, DummyInputs(input1="input1-default", input2=123))
        self.assertEqual(spec._outputs, DummyOutputs(output1=None, output2=None))

        extra_spec = ExtraSpec(
            [
                "--inputStr",
                "abc123",
                "--inputInt",
                "123",
                "--inputBool",
                "True",
                "--inputDict",
                '{"key1":"val1"}',
                "--inputList",
                '["str1","str2"]',
            ]
        )
        self.assertEqual(
            extra_spec._inputs,
            AllInputTypes(
                inputStr="abc123",
                inputInt=123,
                inputBool=True,
                inputDict={"key1": "val1"},
                inputList=["str1", "str2"],
                inputOptional="default-string",
                inputOptionalNoDefault=None,
            ),
        )
        self.assertEqual(extra_spec._outputs, NoOutputs())

    def test_validates_spec_wrong_type(self):
        @dataclass(frozen=True)
        class BadInputs(SageMakerComponentBaseInputs):
            badInput: SageMakerComponentInput

        @dataclass
        class BadOutputs(SageMakerComponentBaseOutputs):
            badOutput: SageMakerComponentOutput

        class BadInputsSpec(
            SageMakerComponentSpec[BadInputs, SageMakerComponentBaseOutputs]
        ):
            INPUTS: BadInputs = BadInputs(badInput="abc1234")
            OUTPUTS = {}

        class BadOutputsSpec(
            SageMakerComponentSpec[SageMakerComponentBaseInputs, BadOutputs]
        ):
            INPUTS = {}
            OUTPUTS = BadOutputs(badOutput="abc123")

        with self.assertRaises(ValueError):
            BadInputsSpec([], BadInputs, SageMakerComponentBaseOutputs)
        with self.assertRaises(ValueError):
            BadOutputsSpec([], SageMakerComponentBaseInputs, BadOutputs)

    def test_creates_parser_correctly(self):
        # Use base spec so we can mock INPUTS and OUTPUTS later
        spec = SageMakerComponentSpec(
            [], SageMakerComponentBaseInputs, SageMakerComponentBaseOutputs
        )

        with patch(
            "common.sagemaker_component_spec.argparse.ArgumentParser", MagicMock()
        ) as mock_parser:
            spec.INPUTS = DummySpec.INPUTS
            spec.OUTPUTS = DummySpec.OUTPUTS
            returned_parser = spec._parser

            mock_parser().add_argument.assert_has_calls(
                [
                    call(
                        "--input1",
                        choices=None,
                        default="input1-default",
                        help="The first input.",
                        required=False,
                        type=str,
                    ),
                    call(
                        "--input2",
                        choices=None,
                        default=None,
                        help="The second input.",
                        required=True,
                        type=int,
                    ),
                    call(
                        "--output1_output_path",
                        default="/tmp/output1",
                        help="The first output.",
                        type=str,
                    ),
                    call(
                        "--output2_output_path",
                        default="/tmp/output2",
                        help="The second output.",
                        type=str,
                    ),
                ]
            )

        with patch(
            "common.sagemaker_component_spec.argparse.ArgumentParser", MagicMock()
        ) as mock_parser:
            spec.INPUTS = ExtraSpec.INPUTS
            spec.OUTPUTS = ExtraSpec.OUTPUTS
            returned_parser = spec._parser

            mock_parser().add_argument.assert_has_calls(
                [
                    call(
                        "--inputInt",
                        choices=None,
                        default=None,
                        help="int",
                        required=True,
                        type=int,
                    ),
                    call(
                        "--inputBool",
                        choices=None,
                        default=None,
                        help="bool",
                        required=True,
                        type=bool,
                    ),
                    call(
                        "--inputDict",
                        choices=None,
                        default=None,
                        help="dict",
                        required=True,
                        type=SpecInputParsers.yaml_or_json_dict,
                    ),
                    call(
                        "--inputList",
                        choices=None,
                        default=None,
                        help="list",
                        required=True,
                        type=SpecInputParsers.yaml_or_json_list,
                    ),
                    call(
                        "--inputOptional",
                        choices=None,
                        default="default-string",
                        help="optional",
                        required=False,
                        type=str,
                    ),
                    call(
                        "--inputOptionalNoDefault",
                        choices=None,
                        default=None,
                        help="optional",
                        required=False,
                        type=str,
                    ),
                ]
            )
