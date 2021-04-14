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
import inspect
import json
from typing import Any, Dict, Tuple

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


def runner(cls_name, method_name, resource_name_output_artifact_path, kwargs):
    cls = getattr(aiplatform, cls_name)

    init_args, method_args = split_args(kwargs)

    serialized_args = {INIT_KEY: init_args, METHOD_KEY: method_args}

    for key, param in inspect.signature(cls.__init__).parameters.items():
        if key in serialized_args[INIT_KEY]:
            serialized_args[INIT_KEY][key] = resolve_init_args(
                key, serialized_args[INIT_KEY][key]
            )
            param_type = utils.resolve_annotation(param.annotation)
            deserializer = utils.get_deserializer(param_type)
            if deserializer:
                serialized_args[INIT_KEY][key] = deserializer(
                    serialized_args[INIT_KEY][key]
                )
    # TODO(chavoshi): use logging instead.
    print(serialized_args[INIT_KEY])
    obj = cls(**serialized_args[INIT_KEY]) if serialized_args[INIT_KEY] else cls

    method = getattr(obj, method_name)

    for key, param in inspect.signature(method).parameters.items():
        if key in serialized_args[METHOD_KEY]:
            param_type = utils.resolve_annotation(param.annotation)
            print(key, param_type)
            serialized_args[METHOD_KEY][key] = resolve_input_args(
                serialized_args[METHOD_KEY][key], param_type
            )
            deserializer = utils.get_deserializer(param_type)
            if deserializer:
                serialized_args[METHOD_KEY][key] = deserializer(
                    serialized_args[METHOD_KEY][key]
                )
            else:
                serialized_args[METHOD_KEY][key] = param_type(
                    serialized_args[METHOD_KEY][key]
                )

    print(serialized_args[METHOD_KEY])
    output = method(**serialized_args[METHOD_KEY])
    print(output)

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
