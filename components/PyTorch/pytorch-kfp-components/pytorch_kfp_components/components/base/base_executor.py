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
"""Pipeline Base Executor class."""
import abc
import logging
from six import with_metaclass


class BaseExecutor(with_metaclass(abc.ABCMeta, object)):  # pylint: disable=R0903
    """Pipeline Base Executor abstract class."""

    def __init__(self):
        pass  # pylint: disable=W0107

    @abc.abstractmethod
    def Do(self, input_dict: dict, output_dict: dict, exec_properties: dict):  # pylint: disable=C0103
        """A Do function that does nothing."""
        pass  # pylint: disable=W0107

    def _log_startup(
        self, input_dict: dict, output_dict: dict, exec_properties
    ):
        """Log inputs, outputs, and executor properties in a standard
        format."""
        class_name = self.__class__.__name__
        logging.debug("Starting %s execution.", class_name)
        logging.debug("Inputs for %s are: %s .", class_name, input_dict)
        logging.debug("Outputs for %s are: %s.", class_name, output_dict)
        logging.debug(
            "Execution Properties for %s are: %s",
            class_name, exec_properties)
