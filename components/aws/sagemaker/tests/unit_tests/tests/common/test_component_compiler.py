from tests.unit_tests.tests.common.dummy_spec import DummySpec, ExtraSpec
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


class ComponentCompilerTestCase(unittest.TestCase):
    # These should always match the dummy spec
    DUMMY_IO_ARGS = IOArgs(
        inputs=[
            InputSpec(
                name="input1",
                description="The first input.",
                type="String",
                default="input1-default",
            ),
            InputSpec(name="input2", description="The second input.", type="Integer"),
        ],
        outputs=[
            OutputSpec(name="output1", description="The first output."),
            OutputSpec(name="output2", description="The second output."),
        ],
        args=[
            "--input1",
            ArgumentValueSpec(inputValue="input1"),
            "--input2",
            ArgumentValueSpec(inputValue="input2"),
            "--output1_output_path",
            ArgumentValueSpec(outputPath="output1"),
            "--output2_output_path",
            ArgumentValueSpec(outputPath="output2"),
        ],
    )

    DUMMY_COMPONENT_SPEC = ComponentSpec(
        name="Dummy component",
        description="Dummy description",
        inputs=DUMMY_IO_ARGS.inputs,
        outputs=DUMMY_IO_ARGS.outputs,
        implementation=ImplementationSpec(
            container=ContainerSpec(
                image="my-image:my-tag",
                command=["python3"],
                args=[
                    "fake-path",
                    "--input1",
                    ArgumentValueSpec(inputValue="input1"),
                    "--input2",
                    ArgumentValueSpec(inputValue="input2"),
                    "--output1_output_path",
                    ArgumentValueSpec(outputPath="output1"),
                    "--output2_output_path",
                    ArgumentValueSpec(outputPath="output2"),
                ],
            )
        ),
    )

    EXTRA_IO_ARGS = IOArgs(
        inputs=[
            InputSpec({"name": "inputStr", "description": "str", "type": "String"}),
            InputSpec({"name": "inputInt", "description": "int", "type": "Integer"}),
            InputSpec({"name": "inputBool", "description": "bool", "type": "Bool"}),
            InputSpec({"name": "inputDict", "description": "dict", "type": "JsonObject"}),
            InputSpec({"name": "inputList", "description": "list", "type": "JsonArray"}),
            InputSpec({
                "name": "inputOptional",
                "description": "optional",
                "type": "String",
                "default": "default-string",
            }),
            InputSpec({
                "name": "inputOptionalNoDefault",
                "description": "optional",
                "type": "String",
                "default": "",
            }),
        ],
        outputs=[],
        args=[
            "--inputStr",
            ArgumentValueSpec({"inputValue": "inputStr"}),
            "--inputInt",
            ArgumentValueSpec({"inputValue": "inputInt"}),
            "--inputBool",
            ArgumentValueSpec({"inputValue": "inputBool"}),
            "--inputDict",
            ArgumentValueSpec({"inputValue": "inputDict"}),
            "--inputList",
            ArgumentValueSpec({"inputValue": "inputList"}),
            "--inputOptional",
            ArgumentValueSpec({"inputValue": "inputOptional"}),
            "--inputOptionalNoDefault",
            ArgumentValueSpec({"inputValue": "inputOptionalNoDefault"}),
        ],
    )

    @classmethod
    def setUpClass(cls):
        cls.compiler = SageMakerComponentCompiler()

    def test_create_io_from_component_spec(self):
        response = SageMakerComponentCompiler._create_io_from_component_spec(DummySpec)  # type: ignore

        self.assertEqual(self.DUMMY_IO_ARGS, response)

    def test_create_io_from_component_spec_extra_types(self):
        response = SageMakerComponentCompiler._create_io_from_component_spec(ExtraSpec)  # type: ignore

        self.assertEqual(self.EXTRA_IO_ARGS, response)

    def test_create_component_spec_composes_correctly(self):
        image_uri = "my-image"
        image_tag = "my-tag"
        file_path = "fake-path"

        expected = ComponentSpec(
            name="Dummy component",
            description="Dummy description",
            inputs=self.DUMMY_IO_ARGS.inputs,
            outputs=self.DUMMY_IO_ARGS.outputs,
            implementation=ImplementationSpec(
                container=ContainerSpec(
                    image="my-image:my-tag",
                    command=["python3"],
                    args=[
                        "fake-path",
                        "--input1",
                        ArgumentValueSpec(inputValue="input1"),
                        "--input2",
                        ArgumentValueSpec(inputValue="input2"),
                        "--output1_output_path",
                        ArgumentValueSpec(outputPath="output1"),
                        "--output2_output_path",
                        ArgumentValueSpec(outputPath="output2"),
                    ],
                )
            ),
        )

        with patch(
            "common.component_compiler.SageMakerComponentCompiler._create_io_from_component_spec",
            MagicMock(return_value=self.DUMMY_IO_ARGS),
        ):
            response = SageMakerComponentCompiler._create_component_spec(
                DummyComponent, file_path, image_uri, image_tag
            )

        # Break it up so diffs aren't too large
        self.assertEqual(
            expected.get("implementation").get("container"),
            response.get("implementation").get("container"),
        )
        self.assertEqual(expected.get("implementation"), response.get("implementation"))
        self.assertEqual(expected, response)

    @patch("common.component_compiler.yaml")
    @patch("builtins.open")
    def test_write_component(self, open_mock, yaml_mock):
        SageMakerComponentCompiler._write_component(DummyComponent, "/tmp/fake-path")

        open_mock.assert_called_once_with("/tmp/fake-path", "w")
        yaml_mock.dump.assert_called_once_with(DummyComponent, ANY)
