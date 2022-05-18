# Copyright 2021-2022 The Kubeflow Authors
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
import inspect
import json
import os
import re
from typing import Any, Callable, Dict, List, Optional, Union

from kfp.component_executor.shared import task_final_status
from kfp.component_executor.shared import artifact_types
from kfp.component_executor.shared import type_annotations


class Executor:
    """Executes Python function components."""

    def __init__(self, executor_input: Dict,
                 function_to_execute: Callable) -> None:
        if hasattr(function_to_execute, 'python_func'):
            self._func = function_to_execute.python_func
        else:
            self._func = function_to_execute

        self._input = executor_input
        self._input_artifacts: Dict[str, artifact_types.Artifact] = {}
        self._output_artifacts: Dict[str, artifact_types.Artifact] = {}

        for name, artifacts in self._input.get('inputs',
                                               {}).get('artifacts', {}).items():
            artifacts_list = artifacts.get('artifacts')
            if artifacts_list:
                self._input_artifacts[name] = self.make_input_artifact(
                    artifacts_list[0])

        for name, artifacts in self._input.get('outputs',
                                               {}).get('artifacts', {}).items():
            artifacts_list = artifacts.get('artifacts')
            if artifacts_list:
                self._output_artifacts[name] = self.make_output_artifact(
                    artifacts_list[0])

        self._return_annotation = inspect.signature(
            self._func).return_annotation
        self._executor_output: Dict[str, Any] = {}

    @classmethod
    def make_input_artifact(cls,
                            runtime_artifact: Dict) -> artifact_types.Artifact:
        return create_runtime_artifact(runtime_artifact)

    @classmethod
    def make_output_artifact(cls,
                             runtime_artifact: Dict) -> artifact_types.Artifact:
        artifact = create_runtime_artifact(runtime_artifact)
        os.makedirs(os.path.dirname(artifact.path), exist_ok=True)
        return artifact

    def get_input_artifact(self, name: str) -> artifact_types.Artifact:
        return self._input_artifacts.get(name)

    def get_output_artifact(self, name: str) -> artifact_types.Artifact:
        return self._output_artifacts.get(name)

    def get_input_parameter_value(self, parameter_name: str):
        parameter_values = self._input.get('inputs',
                                           {}).get('parameterValues', None)

        if parameter_values is not None:
            return parameter_values.get(parameter_name, None)

        return None

    def get_output_parameter_path(self,
                                  parameter_name: str) -> Union[str, None]:
        parameter = self._input.get('outputs',
                                    {}).get('parameters',
                                            {}).get(parameter_name, None)
        if parameter is None:
            return None

        path = parameter.get('outputFile', None)
        if path:
            os.makedirs(os.path.dirname(path), exist_ok=True)
        return path

    def get_output_artifact_path(self, artifact_name: str) -> str:
        output_artifact = self._output_artifacts.get(artifact_name)
        if not output_artifact:
            raise ValueError(
                f'Failed to get output artifact path for artifact name {artifact_name}'
            )
        return output_artifact.path

    def get_input_artifact_path(self, artifact_name: str) -> str:
        input_artifact = self._input_artifacts.get(artifact_name)
        if not input_artifact:
            raise ValueError(
                f'Failed to get input artifact path for artifact name {artifact_name}'
            )
        return input_artifact.path

    def write_output_parameter_value(
            self, name: str, value: Union[str, int, float, bool, dict, list,
                                          Dict, List]) -> None:
        if isinstance(value, (float, int)):
            output = str(value)
        elif isinstance(value, str):
            # value is already a string.
            output = value
        elif isinstance(value, (bool, list, dict)):
            output = json.dumps(value)
        else:
            raise ValueError(
                f'Unable to serialize unknown type `{value}` for parameter input with value `{type(value)}`'
            )

        if not self._executor_output.get('parameterValues'):
            self._executor_output['parameterValues'] = {}

        self._executor_output['parameterValues'][name] = value

    def write_output_artifact_payload(self, name: str, value: Any) -> None:
        path = self.get_output_artifact_path(name)
        with open(path, 'w') as f:
            f.write(str(value))

    @classmethod
    def is_parameter(cls, annotation: Any) -> bool:
        if type(annotation) == type:
            return annotation in [str, int, float, bool, dict, list]

        # Annotation could be, for instance `typing.Dict[str, str]`, etc.
        return get_short_type_name(str(annotation)) in ['Dict', 'List']

    @classmethod
    def is_artifact(cls, annotation: Any) -> bool:
        if type(annotation) == type:
            return issubclass(annotation, artifact_types.Artifact)
        return False

    @classmethod
    def is_named_tuple(cls, annotation: Any) -> bool:
        if type(annotation) == type:
            return issubclass(annotation, tuple) and hasattr(
                annotation, '_fields') and hasattr(annotation,
                                                   '__annotations__')
        return False

    def handle_single_return_value(self, output_name: str, annotation_type: Any,
                                   return_value: Any) -> None:
        if self.is_parameter(annotation_type):
            origin_type = getattr(annotation_type, '__origin__',
                                  None) or annotation_type
            if not isinstance(return_value, origin_type):
                raise ValueError(
                    f'Function `{self._func.__name__}` returned value of type {type(return_value)}; want type {origin_type}'
                )
            self.write_output_parameter_value(output_name, return_value)
        elif self.is_artifact(annotation_type):
            self.write_output_artifact_payload(output_name, return_value)
        else:
            raise RuntimeError(
                f'Unknown return type: {annotation_type}. Must be one of `str`, `int`, `float`, or a subclass of `Artifact`'
            )

    def write_executor_output(self, func_output: Optional[Any] = None) -> None:
        if self._output_artifacts:
            self._executor_output['artifacts'] = {}

        for name, artifact in self._output_artifacts.items():
            runtime_artifact = {
                'name': artifact.name,
                'uri': artifact.uri,
                'metadata': artifact.metadata,
            }
            artifacts_list = {'artifacts': [runtime_artifact]}

            self._executor_output['artifacts'][name] = artifacts_list

        if func_output is not None:
            if self.is_parameter(self._return_annotation) or self.is_artifact(
                    self._return_annotation):
                # Note: single output is named `Output` in component.yaml.
                self.handle_single_return_value('Output',
                                                self._return_annotation,
                                                func_output)
            elif self.is_named_tuple(self._return_annotation):
                if len(self._return_annotation._fields) != len(func_output):
                    raise RuntimeError(
                        f'Expected {len(self._return_annotation._fields)} return values from function `{self._func.__name__}`, got {len(func_output)}'
                    )
                for i in range(len(self._return_annotation._fields)):
                    field = self._return_annotation._fields[i]
                    field_type = self._return_annotation.__annotations__[field]
                    if type(func_output) == tuple:
                        field_value = func_output[i]
                    else:
                        field_value = getattr(func_output, field)
                    self.handle_single_return_value(field, field_type,
                                                    field_value)
            else:
                raise RuntimeError(
                    f'Unknown return type: {self._return_annotation}. Must be one of `str`, `int`, `float`, a subclass of `Artifact`, or a NamedTuple collection of these types.'
                )

        os.makedirs(
            os.path.dirname(self._input['outputs']['outputFile']),
            exist_ok=True)
        with open(self._input['outputs']['outputFile'], 'w') as f:
            f.write(json.dumps(self._executor_output))

    def execute(self) -> None:
        annotations = inspect.getfullargspec(self._func).annotations

        # Function arguments.
        func_kwargs = {}

        for k, v in annotations.items():
            if k == 'return':
                continue

            # Annotations for parameter types could be written as, for example,
            # `Optional[str]`. In this case, we need to strip off the part
            # `Optional[]` to get the actual parameter type.
            v = type_annotations.maybe_strip_optional_from_annotation(v)

            if v == task_final_status.PipelineTaskFinalStatus:
                value = self.get_input_parameter_value(k)
                func_kwargs[k] = task_final_status.PipelineTaskFinalStatus(
                    state=value.get('state'),
                    pipeline_job_resource_name=value.get(
                        'pipelineJobResourceName'),
                    pipeline_task_name=value.get('pipelineTaskName'),
                    error_code=value.get('error').get('code', None),
                    error_message=value.get('error').get('message', None),
                )

            elif self.is_parameter(v):
                value = self.get_input_parameter_value(k)
                if value is not None:
                    func_kwargs[k] = value

            elif type_annotations.is_artifact_annotation(v):
                if type_annotations.is_input_artifact(v):
                    func_kwargs[k] = self.get_input_artifact(k)
                if type_annotations.is_output_artifact(v):
                    func_kwargs[k] = self.get_output_artifact(k)

            elif isinstance(v, type_annotations.OutputPath):
                if self.is_parameter(v.type):
                    func_kwargs[k] = self.get_output_parameter_path(k)
                else:
                    func_kwargs[k] = self.get_output_artifact_path(k)

            elif isinstance(v, type_annotations.InputPath):
                func_kwargs[k] = self.get_input_artifact_path(k)

        result = self._func(**func_kwargs)
        self.write_executor_output(result)


_SCHEMA_TITLE_TO_TYPE: Dict[str, artifact_types.Artifact] = {
    x.TYPE_NAME: x for x in [
        artifact_types.Artifact,
        artifact_types.Model,
        artifact_types.Dataset,
        artifact_types.Metrics,
        artifact_types.ClassificationMetrics,
        artifact_types.SlicedClassificationMetrics,
        artifact_types.HTML,
        artifact_types.Markdown,
    ]
}


def create_runtime_artifact(runtime_artifact: Dict) -> artifact_types.Artifact:
    """Creates an Artifact instance from the specified RuntimeArtifact.

    Args:
        runtime_artifact (Dict): Dictionary representing JSON-encoded RuntimeArtifact.

    Returns:
        artifact_types.Artifact: Artifact instance.
    """
    schema_title = runtime_artifact.get('type', {}).get('schemaTitle', '')

    artifact_type = _SCHEMA_TITLE_TO_TYPE.get(schema_title)
    if not artifact_type:
        artifact_type = artifact_types.Artifact
    return artifact_type(
        uri=runtime_artifact.get('uri', ''),
        name=runtime_artifact.get('name', ''),
        metadata=runtime_artifact.get('metadata', {}),
    )


def get_short_type_name(type_name: str) -> str:
    """Extracts the short form type name.

    This method is used for looking up serializer for a given type.

    For example:
        typing.List -> List
        typing.List[int] -> List
        typing.Dict[str, str] -> Dict
        List -> List
        str -> str

    Args:
        type_name: The original type name.

    Returns:
        The short form type name or the original name if pattern doesn't match.
    """
    match = re.match(r'(typing\.)?(?P<type>\w+)(?:\[.+\])?', type_name)
    return match['type'] if match else type_name
