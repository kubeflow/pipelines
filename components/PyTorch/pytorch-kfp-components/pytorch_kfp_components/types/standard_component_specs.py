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
#pylint: disable=duplicate-code
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
MAR_CONFIG_MODEL_NAME = "model_name"
MAR_CONFIG_MODEL_FILE = "model_file"
MAR_CONFIG_MODEL_HANDLER = "handler"
MAR_CONFIG_SERIALIZED_FILE = "serialized_file"
MAR_CONFIG_VERSION = "version"
MAR_CONFIG_EXPORT_PATH = "export_path"
MAR_CONFIG_CONFIG_PROPERTIES = "config_properties"
MAR_CONFIG_REQUIREMENTS_FILE = "requirements_file"
MAR_CONFIG_EXTRA_FILES = "extra_files"

VIZ_MLPIPELINE_UI_METADATA = "mlpipeline_ui_metadata"
VIZ_MLPIPELINE_METRICS = "mlpipeline_metrics"
VIZ_CONFUSION_MATRIX_DICT = "confusion_matrix_dict"
VIZ_TEST_ACCURACY = "test_accuracy"

VIZ_MARKDOWN = "markdown"
VIZ_MARKDOWN_DICT_SOURCE = "source"
VIZ_MARKDOWN_DICT_STORAGE = "storage"

VIZ_CONFUSION_MATRIX_ACTUALS = "actuals"
VIZ_CONFUSION_MATRIX_PREDS = "preds"
VIZ_CONFUSION_MATRIX_CLASSES = "classes"
VIZ_CONFUSION_MATRIX_URL = "url"

MINIO_SOURCE = "source"
MINIO_BUCKET_NAME = "bucket_name"
MINIO_DESTINATION = "destination"
MINIO_ENDPOINT = "endpoint"

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
    """Mar Specification class.
    For validating the parameter 'type' .
    """

    INPUT_DICT = {
        MAR_GENERATION_CONFIG: Parameters(type=dict),
    }

    OUTPUT_DICT = {}

    EXECUTION_PROPERTIES = {
        MAR_GENERATION_SAVE_PATH: Parameters(type=str, optional=True),
    }

    MAR_CONFIG_DICT = {
        MAR_CONFIG_MODEL_NAME: Parameters(type=str),
        MAR_CONFIG_MODEL_FILE: Parameters(type=str),
        MAR_CONFIG_MODEL_HANDLER: Parameters(type=str),
        MAR_CONFIG_SERIALIZED_FILE: Parameters(type=str),
        MAR_CONFIG_VERSION: Parameters(type=str),
        MAR_CONFIG_EXPORT_PATH: Parameters(type=str, optional=True),
        MAR_CONFIG_CONFIG_PROPERTIES: Parameters(type=str, optional=True),
        MAR_CONFIG_REQUIREMENTS_FILE: Parameters(type=str, optional=True),
        MAR_CONFIG_EXTRA_FILES: Parameters(type=str, optional=True),
    }


class VisualizationSpec:  #pylint: disable=too-few-public-methods
    """Visualization Specification class.
    For validating the parameter 'type'
    """
    INPUT_DICT = {
        VIZ_CONFUSION_MATRIX_DICT: Parameters(type=dict, optional=True),
        VIZ_TEST_ACCURACY: Parameters(type=float, optional=True),
        VIZ_MARKDOWN: Parameters(type=dict, optional=True),
    }

    OUTPUT_DICT = {}

    EXECUTION_PROPERTIES = {
        VIZ_MLPIPELINE_UI_METADATA: Parameters(type=str, optional=True),
        VIZ_MLPIPELINE_METRICS: Parameters(type=str, optional=True),
    }

    MARKDOWN_DICT = {
        VIZ_MARKDOWN_DICT_STORAGE: Parameters(type=str, optional=False),
        VIZ_MARKDOWN_DICT_SOURCE: Parameters(type=dict, optional=False),
    }

    CONFUSION_MATRIX_DICT = {
        VIZ_CONFUSION_MATRIX_ACTUALS: Parameters(type=list, optional=False),
        VIZ_CONFUSION_MATRIX_PREDS: Parameters(type=list, optional=False),
        VIZ_CONFUSION_MATRIX_CLASSES: Parameters(type=list, optional=False),
        VIZ_CONFUSION_MATRIX_URL: Parameters(type=str, optional=False),
    }


class MinIoSpec:  #pylint: disable=too-few-public-methods
    """MinIO Specification class.
    For validating the parameter 'type'
    """
    INPUT_DICT = {
        MINIO_SOURCE: Parameters(type=str),
        MINIO_BUCKET_NAME: Parameters(type=str),
        MINIO_DESTINATION: Parameters(type=str),
    }

    OUTPUT_DICT = {}

    EXECUTION_PROPERTIES = {
        MINIO_ENDPOINT: Parameters(type=str),
    }
