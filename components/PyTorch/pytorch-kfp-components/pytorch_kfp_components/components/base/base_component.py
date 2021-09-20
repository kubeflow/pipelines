#!/usr/bin/env/python3
#
# Copyright (c) Facebook, Inc. and its affiliates.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Pipeline Base component class."""

import abc
from pytorch_kfp_components.types import standard_component_specs


class BaseComponent(metaclass=abc.ABCMeta):  # pylint: disable=R0903
    """Pipeline Base component class."""

    def __init__(self):
        pass

    @classmethod
    def _validate_spec(
        cls,
        spec: standard_component_specs,
        input_dict: dict,
        output_dict: dict,
        exec_properties: dict,
    ):
        """validate the specifications 'type'.

        Args:
            spec: The standard component specifications
            input_dict : A dictionary of inputs.
            ouput-dict :
            exec_properties : A dict of execution properties.
        """

        for key, value in input_dict.items():
            cls._type_check(
                actual_value=value, key=key, spec_dict=spec.INPUT_DICT
            )

        for key, value in output_dict.items():
            cls._type_check(
                actual_value=value, key=key, spec_dict=spec.OUTPUT_DICT
            )

        for key, value in exec_properties.items():
            cls._type_check(
                actual_value=value,
                key=key,
                spec_dict=spec.EXECUTION_PROPERTIES
            )

    @classmethod
    def _optional_check(cls, actual_value: any, key: str, spec_dict: dict):
        """Checks for optional specification.

        Args:
            actual_value : Value of the dictionary.
            key: key for the correspondin value.
            spec_dict : The dict of specification for validation.
        Returns :
            is_optional : The optional key.
        Raises :
            ValueError : If the key is not optional
        """
        is_optional = spec_dict[key].optional

        if not is_optional and not actual_value:
            raise ValueError(
                "{key} is not optional. Received value: {actual_value}".format(
                    key=key, actual_value=actual_value
                )
            )

        return is_optional

    @classmethod
    def _type_check(cls, actual_value, key, spec_dict):
        """Checks the type of specifactions.

        Args:
            actual_value : Value of the dictionary.
            key: key for the correspondin value.
            spec_dict : The dict of specification for validation.

        Raises :
            TypeError : If key value type does not match expected value type.
        """
        if not actual_value:
            is_optional = cls._optional_check(
                actual_value=actual_value, key=key, spec_dict=spec_dict
            )
            if is_optional:
                return

        expected_type = spec_dict[key].type
        actual_type = type(actual_value)
        if actual_type != expected_type:
            raise TypeError(
                "{key} must be of type {expected_type} "
                "but received as {actual_type}"
                .format(
                    key=key,
                    expected_type=expected_type,
                    actual_type=actual_type,
                )
            )
