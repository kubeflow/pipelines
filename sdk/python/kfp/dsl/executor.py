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
import re
from typing import Any, Callable, Dict, List, Optional, Union
import warnings

from kfp import dsl
from kfp.dsl import task_final_status
from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_annotations


class Executor:
    """Executor executes Python function components."""

    def __init__(
        self,
        executor_input: Dict,
        function_to_execute: Union[Callable,
                                   'python_component.PythonComponent'],
    ):

        if hasattr(function_to_execute, 'python_func'):
            self.func = function_to_execute.python_func
        else:
            self.func = function_to_execute

        self.executor_input = executor_input
        self.executor_output_path = self.executor_input['outputs']['outputFile']

        # drop executor_output.json part from the outputFile path
        artifact_types.CONTAINER_TASK_ROOT = os.path.split(
            self.executor_output_path)[0]

        self.input_artifacts: Dict[str, Union[dsl.Artifact,
                                              List[dsl.Artifact]]] = {}
        self.output_artifacts: Dict[str, dsl.Artifact] = {}
        self.assign_input_and_output_artifacts()

        self.return_annotation = inspect.signature(self.func).return_annotation
        self.excutor_output = {}

    def assign_input_and_output_artifacts(self) -> None:
        for name, artifacts in self.executor_input.get('inputs',
                                                       {}).get('artifacts',
                                                               {}).items():
            list_of_artifact_proto_structs = artifacts.get('artifacts')
            if list_of_artifact_proto_structs:
                annotation = self.func.__annotations__[name]
                # InputPath has no attribute __origin__ and also should be handled as a single artifact
                annotation = type_annotations.maybe_strip_optional_from_annotation(
                    annotation)
                is_list_of_artifacts = (
                    type_annotations.is_Input_Output_artifact_annotation(
                        annotation) and
                    type_annotations.is_list_of_artifacts(annotation.__origin__)
                ) or type_annotations.is_list_of_artifacts(annotation)
                if is_list_of_artifacts:
                    # Get the annotation of the inner type of the list
                    # to use when creating the artifacts
                    inner_annotation = type_annotations.get_inner_type(
                        annotation)

                    self.input_artifacts[name] = [
                        self.make_artifact(
                            msg,
                            name,
                            self.func,
                            annotation=inner_annotation,
                        ) for msg in list_of_artifact_proto_structs
                    ]
                else:
                    self.input_artifacts[name] = self.make_artifact(
                        list_of_artifact_proto_structs[0],
                        name,
                        self.func,
                    )

        for name, artifacts in self.executor_input.get('outputs',
                                                       {}).get('artifacts',
                                                               {}).items():
            list_of_artifact_proto_structs = artifacts.get('artifacts')
            if list_of_artifact_proto_structs:
                output_artifact = self.make_artifact(
                    list_of_artifact_proto_structs[0],
                    name,
                    self.func,
                )
                self.output_artifacts[name] = output_artifact
                makedirs_recursively(output_artifact.path)

    def make_artifact(
        self,
        runtime_artifact: Dict,
        name: str,
        func: Callable,
        annotation: Optional[Any] = None,
    ) -> Any:
        annotation = func.__annotations__.get(
            name) if annotation is None else annotation
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
            runtime_artifact, fallback_artifact_cls=artifact_cls)

    def get_input_artifact(self, name: str) -> Optional[dsl.Artifact]:
        return self.input_artifacts.get(name)

    def get_output_artifact(self, name: str) -> Optional[dsl.Artifact]:
        return self.output_artifacts.get(name)

    def get_input_parameter_value(self, parameter_name: str) -> Optional[str]:
        parameter_values = self.executor_input.get('inputs', {}).get(
            'parameterValues', None)

        if parameter_values is not None:
            return parameter_values.get(parameter_name, None)

        return None

    def get_output_parameter_path(self, parameter_name: str) -> Optional[str]:
        parameter = self.executor_input.get('outputs', {}).get(
            'parameters', {}).get(parameter_name, None)
        if parameter is None:
            return None

        path = parameter.get('outputFile', None)
        if path:
            makedirs_recursively(path)
        return path

    def get_output_artifact_path(self, artifact_name: str) -> str:
        output_artifact = self.output_artifacts.get(artifact_name)
        if not output_artifact:
            raise ValueError(
                f'Failed to get output artifact path for artifact name {artifact_name}'
            )
        return output_artifact.path

    def get_input_artifact_path(self, artifact_name: str) -> str:
        input_artifact = self.input_artifacts.get(artifact_name)
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

        if not self.excutor_output.get('parameterValues'):
            self.excutor_output['parameterValues'] = {}

        self.excutor_output['parameterValues'][name] = value

    def write_output_artifact_payload(self, name: str, value: Any) -> None:
        path = self.get_output_artifact_path(name)
        with open(path, 'w') as f:
            f.write(str(value))

    def handle_single_return_value(self, output_name: str, annotation_type: Any,
                                   return_value: Any) -> None:
        if is_parameter(annotation_type):
            origin_type = getattr(annotation_type, '__origin__',
                                  None) or annotation_type
            # relax float-typed return to allow both int and float.
            if origin_type == float:
                accepted_types = (int, float)
            # TODO: relax str-typed return to allow all primitive types?
            else:
                accepted_types = origin_type
            if not isinstance(return_value, accepted_types):
                raise ValueError(
                    f'Function `{self.func.__name__}` returned value of type {type(return_value)}; want type {origin_type}'
                )
            self.write_output_parameter_value(output_name, return_value)

        elif is_artifact(annotation_type):
            if isinstance(return_value, artifact_types.Artifact):
                # for -> Artifact annotations, where the user returns an artifact
                artifact_name = self.executor_input['outputs']['artifacts'][
                    output_name]['artifacts'][0]['name']
                # users should not override the name for Vertex Pipelines
                # if empty string, replace
                # else provide descriptive warning and prefer letting backend throw exception
                running_on_vertex = 'VERTEX_AI_PIPELINES_RUN_LABELS' in os.environ
                if running_on_vertex:
                    if return_value.name == '':
                        return_value.name = artifact_name
                    else:
                        # prefer letting the backend throw the runtime exception
                        warnings.warn(
                            f'If you are running your pipeline Vertex AI Pipelines, you should not provide a name for your artifact. It will be set to the Vertex artifact resource name {artifact_name} by default. Got value for name: {return_value.name}.',
                            RuntimeWarning,
                            stacklevel=2)
                self.output_artifacts[output_name] = return_value
            else:
                # for -> Artifact annotations, where the user returns some data that the executor should serialize
                self.write_output_artifact_payload(output_name, return_value)
        else:
            raise RuntimeError(
                f'Unknown return type: {annotation_type}. Must be one of the supported data types: https://www.kubeflow.org/docs/components/pipelines/v2/data-types/'
            )

    def write_executor_output(self,
                              func_output: Optional[Any] = None
                             ) -> Optional[str]:
        """Writes executor output containing the Python function output. The
        executor output file will not be written if this code is executed from
        a non-chief node in a mirrored execution strategy.

        Args:
            func_output: The object returned by the function.

        Returns:
            Optional[str]: Returns the location of the executor_output file as a string if the file is written. Else, None.
        """

        if func_output is not None:
            if is_parameter(self.return_annotation) or is_artifact(
                    self.return_annotation):
                # Note: single output is named `Output` in component.yaml.
                self.handle_single_return_value('Output',
                                                self.return_annotation,
                                                func_output)
            elif is_named_tuple(self.return_annotation):
                if len(self.return_annotation._fields) != len(func_output):
                    raise RuntimeError(
                        f'Expected {len(self.return_annotation._fields)} return values from function `{self.func.__name__}`, got {len(func_output)}'
                    )
                for i in range(len(self.return_annotation._fields)):
                    field = self.return_annotation._fields[i]
                    field_type = self.return_annotation.__annotations__[field]
                    if type(func_output) == tuple:
                        field_value = func_output[i]
                    else:
                        field_value = getattr(func_output, field)
                    self.handle_single_return_value(field, field_type,
                                                    field_value)
            else:
                raise RuntimeError(
                    f'Unknown return type: {self.return_annotation}. Must be one of `str`, `int`, `float`, a subclass of `Artifact`, or a NamedTuple collection of these types.'
                )

        if self.output_artifacts:
            self.excutor_output['artifacts'] = {}

        for name, artifact in self.output_artifacts.items():
            runtime_artifact = {
                'name': artifact.name,
                'uri': artifact.uri,
                'metadata': artifact.metadata,
            }
            artifacts_list = {'artifacts': [runtime_artifact]}

            self.excutor_output['artifacts'][name] = artifacts_list

        # This check is to ensure only one worker (in a mirrored, distributed training/compute strategy) attempts to write to the same executor output file at the same time using gcsfuse, which enforces immutability of files.
        write_file = True

        CLUSTER_SPEC_ENV_VAR_NAME = 'CLUSTER_SPEC'
        cluster_spec_string = os.environ.get(CLUSTER_SPEC_ENV_VAR_NAME)
        if cluster_spec_string:
            cluster_spec = json.loads(cluster_spec_string)
            CHIEF_NODE_LABELS = {'workerpool0', 'chief', 'master'}
            write_file = cluster_spec['task']['type'] in CHIEF_NODE_LABELS

        if write_file:
            makedirs_recursively(self.executor_output_path)
            with open(self.executor_output_path, 'w') as f:
                f.write(json.dumps(self.excutor_output))
            return self.executor_output_path
        return None

    def execute(self) -> Optional[str]:
        """Executes the function and writes the executor output file. The
        executor output file will not be written if this code is executed from
        a non-chief node in a mirrored execution strategy.

        Returns:
            Optional[str]: Returns the location of the executor_output file as a string if the file is written. Else, None.
        """
        annotations = inspect.getfullargspec(self.func).annotations

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

                # PipelineTaskFinalStatus field names pipelineJobResourceName and pipelineTaskName are deprecated. Support for these fields will be removed at a later date.
                pipline_job_resource_name = 'pipelineJobResourceName'
                if value.get(pipline_job_resource_name) is None:
                    pipline_job_resource_name = 'pipeline_job_resource_name'
                pipeline_task_name = 'pipelineTaskName'
                if value.get(pipeline_task_name) is None:
                    pipeline_task_name = 'pipeline_task_name'

                func_kwargs[k] = task_final_status.PipelineTaskFinalStatus(
                    state=value.get('state'),
                    pipeline_job_resource_name=value.get(
                        pipline_job_resource_name),
                    pipeline_task_name=value.get(pipeline_task_name),
                    error_code=value.get('error', {}).get('code', None),
                    error_message=value.get('error', {}).get('message', None),
                )

            elif type_annotations.is_list_of_artifacts(v):
                func_kwargs[k] = self.get_input_artifact(k)

            elif is_parameter(v):
                value = self.get_input_parameter_value(k)
                if value is not None:
                    func_kwargs[k] = value

            elif type_annotations.is_Input_Output_artifact_annotation(v):
                if type_annotations.is_artifact_wrapped_in_Input(v):
                    func_kwargs[k] = self.get_input_artifact(k)
                if type_annotations.is_artifact_wrapped_in_Output(v):
                    func_kwargs[k] = self.get_output_artifact(k)

            elif is_artifact(v):
                func_kwargs[k] = self.get_input_artifact(k)

            elif isinstance(v, type_annotations.OutputPath):
                if is_parameter(v.type):
                    func_kwargs[k] = self.get_output_parameter_path(k)
                else:
                    func_kwargs[k] = self.get_output_artifact_path(k)

            elif isinstance(v, type_annotations.InputPath):
                func_kwargs[k] = self.get_input_artifact_path(k)

        result = self.func(**func_kwargs)
        return self.write_executor_output(result)


def create_artifact_instance(
    runtime_artifact: Dict,
    fallback_artifact_cls=dsl.Artifact,
) -> type:
    """Creates an artifact class instances from a runtime artifact
    dictionary."""
    schema_title = runtime_artifact.get('type', {}).get('schemaTitle', '')
    artifact_cls = artifact_types._SCHEMA_TITLE_TO_TYPE.get(
        schema_title, fallback_artifact_cls)
    return artifact_cls._from_executor_fields(
        uri=runtime_artifact.get('uri', ''),
        name=runtime_artifact.get('name', ''),
        metadata=runtime_artifact.get('metadata', {}),
    ) if hasattr(artifact_cls, '_from_executor_fields') else artifact_cls(
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


# TODO: merge with type_utils.is_parameter_type
def is_parameter(annotation: Any) -> bool:
    if type(annotation) == type:
        return annotation in [str, int, float, bool, dict, list]

    # Annotation could be, for instance `typing.Dict[str, str]`, etc.
    return get_short_type_name(str(annotation)) in ['Dict', 'List']


def is_artifact(annotation: Any) -> bool:
    if type(annotation) == type:
        return type_annotations.is_artifact_class(annotation)
    return False


def is_named_tuple(annotation: Any) -> bool:
    if type(annotation) == type:
        return issubclass(annotation, tuple) and hasattr(
            annotation, '_fields') and hasattr(annotation, '__annotations__')
    return False


def makedirs_recursively(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
