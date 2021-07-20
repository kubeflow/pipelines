# !/usr/bin/env/python3
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
# pylint: disable=no-self-use,too-many-arguments,unused-argument,not-callable
""" Utilities for parsing arguments - Samples"""


def parse_input_args(input_str: str):
    """
    Utility to parse input string arguments. Returns a dictionary
    """
    output_dict = {}
    if not input_str:
        raise ValueError("Empty input string: {}".format(input_str))

    key_pairs: list = input_str.split(",")

    key_pairs = [x.strip() for x in key_pairs]

    if not key_pairs:
        raise ValueError("Incorrect format: {}".format(input_str))

    for each_key in key_pairs:
        try:
            key, value = each_key.split("=")
        except ValueError as value_error:
            raise ValueError("Expected input format "
                             "'key1=value1, key2=value2' "
                             "but received {}".format(input_str)) \
                from value_error
        if value.isdigit():
            value = int(value)
        output_dict[key] = value

    return output_dict
