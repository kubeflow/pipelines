from common.spec_input_parsers import SpecInputParsers
from common.sagemaker_component_spec import SageMakerComponentSpec

from typing import List
from dataclasses import dataclass

from common.sagemaker_component_spec import SageMakerComponentSpec
from common.common_inputs import (
    SageMakerComponentBaseInputs,
    SageMakerComponentBaseOutputs,
    SageMakerComponentInputValidator as InputValidator,
    SageMakerComponentOutputValidator as OutputValidator,
    SageMakerComponentInput as Input,
    SageMakerComponentOutput as Output,
)


@dataclass(frozen=True)
class DummyInputs(SageMakerComponentBaseInputs):
    input1: Input
    input2: Input


@dataclass
class DummyOutputs(SageMakerComponentBaseOutputs):
    output1: Output
    output2: Output


@dataclass(frozen=True)
class AllInputTypes(SageMakerComponentBaseInputs):
    inputStr: Input
    inputInt: Input
    inputBool: Input
    inputDict: Input
    inputList: Input
    inputOptional: Input
    inputOptionalNoDefault: Input


@dataclass
class NoOutputs(SageMakerComponentBaseOutputs):
    pass


class DummySpec(SageMakerComponentSpec[DummyInputs, DummyOutputs]):
    INPUTS: DummyInputs = DummyInputs(
        input1=InputValidator(
            input_type=str, description="The first input.", default="input1-default",
        ),
        input2=InputValidator(
            input_type=int, required=True, description="The second input.",
        ),
    )

    OUTPUTS = DummyOutputs(
        output1=OutputValidator(description="The first output."),
        output2=OutputValidator(description="The second output."),
    )

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, DummyInputs, DummyOutputs)


class ExtraSpec(SageMakerComponentSpec[AllInputTypes, NoOutputs]):
    INPUTS: AllInputTypes = AllInputTypes(
        inputStr=InputValidator(input_type=str, required=True, description="str",),
        inputInt=InputValidator(input_type=int, required=True, description="int",),
        inputBool=InputValidator(input_type=bool, required=True, description="bool",),
        inputDict=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_dict,
            required=True,
            description="dict",
        ),
        inputList=InputValidator(
            input_type=SpecInputParsers.yaml_or_json_list,
            required=True,
            description="list",
        ),
        inputOptional=InputValidator(
            input_type=str,
            required=False,
            description="optional",
            default="default-string",
        ),
        inputOptionalNoDefault=InputValidator(
            input_type=str, required=False, description="optional",
        ),
    )

    OUTPUTS = NoOutputs()

    def __init__(self, arguments: List[str]):
        super().__init__(arguments, AllInputTypes, NoOutputs)
