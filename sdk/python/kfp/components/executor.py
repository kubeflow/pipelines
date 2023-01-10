# Copyright 2021 The Kubeflow Authors
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
from typing import Any, Callable, Dict, List, Optional, Union

from kfp.components import task_final_status
from kfp.components.types import artifact_types
from kfp.components.types import type_annotations


class Executor():
    """Executor executes v2-based Python function components."""

    def __init__(self, executor_input: Dict, function_to_execute: Callable):
        self._func = function_to_execute

        self._input = executor_input
        self._input_artifacts: Dict[str,
                                    Union[artifact_types.Artifact,
                                          List[artifact_types.Artifact]]] = {}
        self._output_artifacts: Dict[str, artifact_types.Artifact] = {}

        for name, artifacts in self._input.get('inputs',
                                               {}).get('artifacts', {}).items():
            list_of_artifact_proto_structs = artifacts.get('artifacts')
            if list_of_artifact_proto_structs:
                annotation = self._func.__annotations__[name]
                # InputPath has no attribute __origin__ and also should be handled as a single artifact
                if type_annotations.is_Input_Output_artifact_annotation(
                        annotation) and type_annotations.is_list_of_artifacts(
                            annotation.__origin__):
                    self._input_artifacts[name] = [
                        self.make_artifact(
                            msg,
                            name,
                            self._func,
                        ) for msg in list_of_artifact_proto_structs
                    ]
                else:
                    self._input_artifacts[name] = self.make_artifact(
                        list_of_artifact_proto_structs[0],
                        name,
                        self._func,
                    )

        for name, artifacts in self._input.get('outputs',
                                               {}).get('artifacts', {}).items():
            list_of_artifact_proto_structs = artifacts.get('artifacts')
            if list_of_artifact_proto_structs:
                output_artifact = self.make_artifact(
                    list_of_artifact_proto_structs[0],
                    name,
                    self._func,
                )
                self._output_artifacts[name] = output_artifact
                self.makedirs_recursively(output_artifact.path)

        self._return_annotation = inspect.signature(
            self._func).return_annotation
        self._executor_output = {}

    def make_artifact(
        self,
        runtime_artifact: Dict,
        name: str,
        func: Callable,
    ) -> Any:
        annotation = func.__annotations__.get(name)
        if isinstance(annotation, type_annotations.InputPath):
            schema_title, _ = annotation.type.split('@')
            if schema_title in artifact_types._SCHEMA_TITLE_TO_TYPE:
                artifact_cls = artifact_types._SCHEMA_TITLE_TO_TYPE[
                    schema_title]
            else:
                raise TypeError(
                    f'Invalid type argument to {type_annotations.InputPath.__name__}: {annotation.type}'
                )
        else:
            artifact_cls = annotation
        return create_artifact_instance(
            runtime_artifact, artifact_cls=artifact_cls)

    def makedirs_recursively(self, path: str) -> None:
        os.makedirs(os.path.dirname(path), exist_ok=True)

    def _get_input_artifact(self, name: str):
        return self._input_artifacts.get(name)

    def _get_output_artifact(self, name: str):
        return self._output_artifacts.get(name)

    def _get_input_parameter_value(self, parameter_name: str):
        parameter_values = self._input.get('inputs',
                                           {}).get('parameterValues', None)

        if parameter_values is not None:
            return parameter_values.get(parameter_name, None)

        return None

    def _get_output_parameter_path(self, parameter_name: str):
        parameter = self._input.get('outputs',
                                    {}).get('parameters',
                                            {}).get(parameter_name, None)
        if parameter is None:
            return None

        import os
        path = parameter.get('outputFile', None)
        if path:
            os.makedirs(os.path.dirname(path), exist_ok=True)
        return path

    def _get_output_artifact_path(self, artifact_name: str):
        output_artifact = self._output_artifacts.get(artifact_name)
        if not output_artifact:
            raise ValueError(
                f'Failed to get output artifact path for artifact name {artifact_name}'
            )
        return output_artifact.path

    def _get_input_artifact_path(self, artifact_name: str):
        input_artifact = self._input_artifacts.get(artifact_name)
        if not input_artifact:
            raise ValueError(
                f'Failed to get input artifact path for artifact name {artifact_name}'
            )
        return input_artifact.path

    def _write_output_parameter_value(self, name: str,
                                      value: Union[str, int, float, bool, dict,
                                                   list, Dict, List]):
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

    def _write_output_artifact_payload(self, name: str, value: Any):
        path = self._get_output_artifact_path(name)
        with open(path, 'w') as f:
            f.write(str(value))

    # TODO: extract to a util
    @classmethod
    def _get_short_type_name(cls, type_name: str) -> str:
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
        import re
        match = re.match('(typing\.)?(?P<type>\w+)(?:\[.+\])?', type_name)
        return match.group('type') if match else type_name

    # TODO: merge with type_utils.is_parameter_type
    @classmethod
    def _is_parameter(cls, annotation: Any) -> bool:
        if type(annotation) == type:
            return annotation in [str, int, float, bool, dict, list]

        # Annotation could be, for instance `typing.Dict[str, str]`, etc.
        return cls._get_short_type_name(str(annotation)) in ['Dict', 'List']

    @classmethod
    def _is_artifact(cls, annotation: Any) -> bool:
        if type(annotation) == type:
            return type_annotations.is_artifact_class(annotation)
        return False

    @classmethod
    def _is_named_tuple(cls, annotation: Any) -> bool:
        if type(annotation) == type:
            return issubclass(annotation, tuple) and hasattr(
                annotation, '_fields') and hasattr(annotation,
                                                   '__annotations__')
        return False

    def _handle_single_return_value(self, output_name: str,
                                    annotation_type: Any, return_value: Any):
        if self._is_parameter(annotation_type):
            origin_type = getattr(annotation_type, '__origin__',
                                  None) or annotation_type
            if not isinstance(return_value, origin_type):
                raise ValueError(
                    f'Function `{self._func.__name__}` returned value of type {type(return_value)}; want type {origin_type}'
                )
            self._write_output_parameter_value(output_name, return_value)
        elif self._is_artifact(annotation_type):
            self._write_output_artifact_payload(output_name, return_value)
        else:
            raise RuntimeError(
                f'Unknown return type: {annotation_type}. Must be one of `str`, `int`, `float`, or a subclass of `Artifact`'
            )

    def _write_executor_output(self, func_output: Optional[Any] = None):
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
            if self._is_parameter(self._return_annotation) or self._is_artifact(
                    self._return_annotation):
                # Note: single output is named `Output` in component.yaml.
                self._handle_single_return_value('Output',
                                                 self._return_annotation,
                                                 func_output)
            elif self._is_named_tuple(self._return_annotation):
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
                    self._handle_single_return_value(field, field_type,
                                                     field_value)
            else:
                raise RuntimeError(
                    f'Unknown return type: {self._return_annotation}. Must be one of `str`, `int`, `float`, a subclass of `Artifact`, or a NamedTuple collection of these types.'
                )

        # This check is to ensure only one worker (in a mirrored, distributed training/compute strategy) attempts to write to the same executor output file at the same time using gcsfuse, which enforces immutability of files.
        write_file = True

        CLUSTER_SPEC_ENV_VAR_NAME = 'CLUSTER_SPEC'
        cluster_spec_string = os.environ.get(CLUSTER_SPEC_ENV_VAR_NAME)
        if cluster_spec_string:
            cluster_spec = json.loads(cluster_spec_string)
            CHIEF_NODE_LABELS = {'workerpool0', 'chief', 'master'}
            write_file = cluster_spec['task']['type'] in CHIEF_NODE_LABELS

        if write_file:
            executor_output_path = self._input['outputs']['outputFile']
            os.makedirs(os.path.dirname(executor_output_path), exist_ok=True)
            with open(executor_output_path, 'w') as f:
                f.write(json.dumps(self._executor_output))

    def execute(self):
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
                value = self._get_input_parameter_value(k)
                func_kwargs[k] = task_final_status.PipelineTaskFinalStatus(
                    state=value.get('state'),
                    pipeline_job_resource_name=value.get(
                        'pipelineJobResourceName'),
                    pipeline_task_name=value.get('pipelineTaskName'),
                    error_code=value.get('error').get('code', None),
                    error_message=value.get('error').get('message', None),
                )

            elif self._is_parameter(v):
                value = self._get_input_parameter_value(k)
                if value is not None:
                    func_kwargs[k] = value

            elif type_annotations.is_Input_Output_artifact_annotation(v):
                if type_annotations.is_input_artifact(v):
                    func_kwargs[k] = self._get_input_artifact(k)
                if type_annotations.is_output_artifact(v):
                    func_kwargs[k] = self._get_output_artifact(k)

            elif isinstance(v, type_annotations.OutputPath):
                if self._is_parameter(v.type):
                    func_kwargs[k] = self._get_output_parameter_path(k)
                else:
                    func_kwargs[k] = self._get_output_artifact_path(k)

            elif isinstance(v, type_annotations.InputPath):
                func_kwargs[k] = self._get_input_artifact_path(k)

        result = self._func(**func_kwargs)
        self._write_executor_output(result)


def create_artifact_instance(
    runtime_artifact: Dict,
    artifact_cls=artifact_types.Artifact,
) -> type:
    """Creates an artifact class instances from a runtime artifact
    dictionary."""
    schema_title = runtime_artifact.get('type', {}).get('schemaTitle', '')

    artifact_cls = artifact_types._SCHEMA_TITLE_TO_TYPE.get(
        schema_title, artifact_cls)
    return artifact_cls._from_executor_fields(
        uri=runtime_artifact.get('uri', ''),
        name=runtime_artifact.get('name', ''),
        metadata=runtime_artifact.get('metadata', {}),
    ) if hasattr(artifact_cls, '_from_executor_fields') else artifact_cls(
        uri=runtime_artifact.get('uri', ''),
        name=runtime_artifact.get('name', ''),
        metadata=runtime_artifact.get('metadata', {}),
    )
