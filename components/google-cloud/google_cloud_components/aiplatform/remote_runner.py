# Copyright 2021 Google LLC. All Rights Reserved.
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
"""Module for remote execution of AI Platform component."""

import argparse
from distutils import util as distutil
import inspect
import json
import os
from typing import Any, Callable, Dict, Tuple, Type, TypeVar

from google.cloud import aiplatform
from google_cloud_components.aiplatform import utils

INIT_KEY = 'init'
METHOD_KEY = 'method'


def split_args(kwargs: Dict[str, Any]) -> Tuple[Dict[str, Any], Dict[str, Any]]:
    """Splits args into constructor and method args.

    Args:
        kwargs: kwargs with parameter names preprended with init or method
    Returns:
        constructor kwargs, method kwargs
    """
    init_args = {}
    method_args = {}

    for key, arg in kwargs.items():
        if key.startswith(INIT_KEY):
            init_args[key.split(".")[-1]] = arg
        elif key.startswith(METHOD_KEY):
            method_args[key.split(".")[-1]] = arg

    return init_args, method_args


def write_to_artifact(artifact_path, text):
    """Write output to local artifact path (uses GCSFuse)."""

    # Create the artifact dir if it does not exist
    os.makedirs(os.path.dirname(artifact_path), exist_ok=True)

    with open(artifact_path, 'w') as f:
        f.write(text)


def read_from_artifact(artifact_path):
    """Read input from local artifact path(uses GCSFuse)."""
    with open(artifact_path, 'r') as input_file:
        resource_name = input_file.read()
    return resource_name


def resolve_input_args(value, type_to_resolve):
    """If this is an input from Pipelines, read it directly from gcs."""
    if inspect.isclass(type_to_resolve) and issubclass(
            type_to_resolve, aiplatform.base.AiPlatformResourceNoun):
        if value.startswith('/gcs/'):  # not a resource noun:
            value = read_from_artifact(value)
    return value


def resolve_init_args(key, value):
    """Resolves Metadata/InputPath parameters to resource names."""
    if key.endswith('_name'):
        if value.startswith('/gcs/'):  # not a resource noun
            value = read_from_artifact(value)
    return value


def make_output(output_object: Any) -> str:
    if utils.is_mb_sdk_resource_noun_type(type(output_object)):
        return output_object.resource_name

    # TODO: handle more default cases
    # right now this is required for export data because proto Repeated
    # this should be expanded to handle multiple different types
    # or possibly export data should return a Dataset
    return json.dumps(list(output_object))


T = TypeVar('T')


def cast(value: str, annotation_type: Type[T]) -> T:
    """Casts a value to the annotation type.

    Includes special handling for bools passed as strings.

    Args:
        value (str): The value represented as a string.
        annotation_type (Type[T]): The type to cast the value to.
    Returns:
        An instance of annotation_type value.
    """
    if annotation_type is bool:
        return bool(distutil.strtobool(value))
    return annotation_type(value)


def prepare_parameters(
    kwargs: Dict[str, Any], method: Callable, is_init: bool = False
):
    """Prepares paramters passed into components before calling SDK.

    1. Determines the annotation type that should used with the parameter
    2. Reads input values if needed
    3. Deserializes thos value where appropriate
    4. Or casts to the correct type.

    Args:
        kwargs (Dict[str, Any]): The kwargs that will be passed into method. Mutates in place.
        method (Callable): The method the kwargs used to invoke the method.
        is_init (bool): Whether this method is a constructor
    """
    for key, param in inspect.signature(method).parameters.items():
        if key in kwargs:
            value = kwargs[key]
            param_type = utils.resolve_annotation(param.annotation)
            value = resolve_init_args(
                key, value
            ) if is_init else resolve_input_args(value, param_type)
            deserializer = utils.get_deserializer(param_type)
            if deserializer:
                value = deserializer(value)
            else:
                value = cast(value, param_type)
            kwargs[key] = value


def runner(cls_name, method_name, resource_name_output_artifact_path, kwargs):
    cls = getattr(aiplatform, cls_name)

    init_args, method_args = split_args(kwargs)

    serialized_args = {INIT_KEY: init_args, METHOD_KEY: method_args}

    prepare_parameters(serialized_args[INIT_KEY], cls.__init__, is_init=True)
    obj = cls(**serialized_args[INIT_KEY]) if serialized_args[INIT_KEY] else cls

    method = getattr(obj, method_name)
    prepare_parameters(serialized_args[METHOD_KEY], method, is_init=False)
    output = method(**serialized_args[METHOD_KEY])

    if output:
        write_to_artifact(
            resource_name_output_artifact_path, make_output(output)
        )
        return output


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cls_name", type=str)
    parser.add_argument("--method_name", type=str)
    parser.add_argument(
        "--resource_name_output_artifact_path", type=str, default=None
    )

    args, unknown_args = parser.parse_known_args()
    kwargs = {}

    key_value = None
    for arg in unknown_args:
        print(arg)
        if "=" in arg:
            key, value = arg[2:].split("=")
            kwargs[key] = value
        else:
            if not key_value:
                key_value = arg[2:]
            else:
                kwargs[key_value] = arg
                key_value = None

    print(
        runner(
            args.cls_name, args.method_name,
            args.resource_name_output_artifact_path, kwargs
        )
    )


if __name__ == "__main__":
    main()
