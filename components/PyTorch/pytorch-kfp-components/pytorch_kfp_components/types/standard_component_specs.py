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
"""Module for defining standard specifications and validation of parameter
type."""

TRAINER_MODULE_FILE = "module_file"
TRAINER_DATA_MODULE_FILE = "data_module_file"
TRAINER_DATA_MODULE_ARGS = "data_module_args"
TRAINER_MODULE_ARGS = "module_file_args"
PTL_TRAINER_ARGS = "trainer_args"

TRAINER_MODEL_SAVE_PATH = "model_save_path"
PTL_TRAINER_OBJ = "ptl_trainer"


MAR_GENERATION_CONFIG = "mar_config"
MAR_GENERATION_SAVE_PATH = "mar_save_path"
CONFIG_PROPERTIES_SAVE_PATH = "config_prop_save_path"


class Parameters:  # pylint: disable=R0903
    """Parameter class to match the desired type."""

    def __init__(self, type=None, optional=False):  # pylint: disable=redefined-builtin
        self.type = type
        self.optional = optional


class TrainerSpec:  # pylint: disable=R0903
    """Trainer Specification class.

    For validating the parameter 'type' .
    """

    INPUT_DICT = {
        TRAINER_MODULE_FILE: Parameters(type=str),
        TRAINER_DATA_MODULE_FILE: Parameters(type=str),
    }

    OUTPUT_DICT = {}

    EXECUTION_PROPERTIES = {
        TRAINER_DATA_MODULE_ARGS: Parameters(type=dict, optional=True),
        TRAINER_MODULE_ARGS: Parameters(type=dict),
        PTL_TRAINER_ARGS: Parameters(type=dict, optional=True),
    }


class MarGenerationSpec:  # pylint: disable=R0903
    """Trainer Specification class.

    For validating the parameter 'type' .
    """

    INPUT_DICT = {
        MAR_GENERATION_CONFIG: Parameters(type=dict),
    }

    OUTPUT_DICT = {}

    EXECUTION_PROPERTIES = {
        MAR_GENERATION_SAVE_PATH: Parameters(type=str, optional=True),
    }
