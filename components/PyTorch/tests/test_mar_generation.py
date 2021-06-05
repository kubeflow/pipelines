#!/usr/bin/env/python3
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
"""Unit Tests for mar generation."""
import os
import re
import subprocess
import tempfile
import pytest
from pytorch_kfp_components.components.mar.component import MarGeneration

IRIS_DIR = "tests/iris"
EXPORT_PATH = tempfile.mkdtemp()
print(f"Export path: {EXPORT_PATH}")
MAR_CONFIG = {
    "MODEL_NAME":
    "iris_classification",
    "MODEL_FILE":
    f"{IRIS_DIR}/iris_classification.py",
    "HANDLER":
    f"{IRIS_DIR}/iris_handler.py",
    "SERIALIZED_FILE":
    f"{EXPORT_PATH}/iris.pt",
    "VERSION":
    "1",
    "EXPORT_PATH":
    EXPORT_PATH,
    "CONFIG_PROPERTIES":
    "https://kubeflow-dataset.s3.us-east-2.amazonaws.com/config.properties",
}

MANDATORY_ARGS = [
    "MODEL_NAME",
    "SERIALIZED_FILE",
    "MODEL_FILE",
    "HANDLER",
    "VERSION",
    "CONFIG_PROPERTIES",
]

OPTIONAL_ARGS = ["EXTRA_FILES", "REQUIREMENTS_FILE"]

DEFAULT_HANDLERS = [
    "image_classifier",
    "text_classifier",
    "image_segmenter",
    "object_detector",
]


def generate_mar_file(config, save_path):
    """Generates a mar file with proper configs
    Args:
        config : mar config dict
        save_path : mar file save path
    """
    MarGeneration(mar_config=config, mar_save_path=save_path)
    mar_path = os.path.join(save_path, "iris_classification.mar")
    config_properties = os.path.join(save_path, "config.properties")
    assert os.path.exists(mar_path)
    assert os.path.exists(config_properties)

    os.remove(mar_path)
    os.remove(config_properties)


def test_invalid_mar_config_parameter_type():
    """Testing mar generation failure with invalid mar configs."""
    mar_config = "invalid_mar_config"
    tmp_dir = tempfile.mkdtemp()

    exception_msg = re.escape(
        f"mar_config must be of "
        f"type <class 'dict'> but received as {type(mar_config)}"
    )
    with pytest.raises(TypeError, match=exception_msg):
        MarGeneration(mar_config=mar_config, mar_save_path=tmp_dir)


def test_invalid_mar_save_parameter_type():
    """Test mar generation failure with invalid save path.

    Raises:
        TypeError: for passing invalid path type
    """
    mar_save_parameter = ["mar_save_path"]
    exception_msg = re.escape(
        f"mar_save_path must be of "
        f"type <class 'str'> but received as {type(mar_save_parameter)}"
    )
    with pytest.raises(TypeError, match=exception_msg):
        MarGeneration(mar_config=MAR_CONFIG, mar_save_path=mar_save_parameter)


def test_invalid_mar_config_parameter_value():
    """Test mar generation failure with empty mar config.

    Raises:
        ValueError: If mar config is empty.
    """
    mar_config = {}
    tmp_dir = tempfile.mkdtemp()

    exception_msg = re.escape(
        "mar_config is not optional. Received value: {}".format(mar_config))
    with pytest.raises(ValueError, match=exception_msg):
        MarGeneration(mar_config=mar_config, mar_save_path=tmp_dir)


@pytest.mark.parametrize("mandatory_key", MANDATORY_ARGS)
def test_mar_generation_mandatory_params_missing(mandatory_key):
    """Testing Mar Generation with missing mandatory keys.

    Args:
        mandatory_key : mandatory keys in mar config
    Raises:
        Exception : when mandatory keys are missing.
    """
    tmp_value = MAR_CONFIG[mandatory_key]
    MAR_CONFIG[mandatory_key] = ""
    MAR_CONFIG.pop(mandatory_key)

    tmp_dir = tempfile.mkdtemp()
    excpetion_msg = re.escape(
        f"Following Mandatory keys are "
        f"missing in the config file ['{mandatory_key}']"
    )
    with pytest.raises(Exception, match=excpetion_msg):
        MarGeneration(mar_config=MAR_CONFIG, mar_save_path=tmp_dir)

    MAR_CONFIG[mandatory_key] = tmp_value


def test_mar_generation_success():
    """Test for successful mar generation."""
    cmd = [
        "python",
        "iris_pytorch.py",
        "--checkpoint_dir",
        EXPORT_PATH,
    ]

    cwd = os.getcwd()
    os.chdir(IRIS_DIR)
    subprocess.run(cmd)  #pylint: disable=W1510
    os.chdir(cwd)
    generate_mar_file(config=MAR_CONFIG, save_path=EXPORT_PATH)
    print("export path need to watch >>>>>>>>>>>>>",EXPORT_PATH)


@pytest.mark.parametrize("handler", DEFAULT_HANDLERS)
def test_mar_generation_default_handlers(handler):
    """Testing mar generation using default handlers.

    Args:
        handler: default handler files
    """
    tmp_value = MAR_CONFIG["HANDLER"]
    MAR_CONFIG["HANDLER"] = handler
    generate_mar_file(config=MAR_CONFIG, save_path=EXPORT_PATH)
    MAR_CONFIG["HANDLER"] = tmp_value


@pytest.mark.parametrize("optional_arg", OPTIONAL_ARGS)
def test_mar_generation_optional_arguments(optional_arg):
    """Tests mar generation with optional arguments.

    Args:
        optional_arg : optional args for mar generation
    """
    new_file, filename = tempfile.mkstemp()  #pylint: disable=W0612

    MAR_CONFIG[optional_arg] = os.path.join(os.getcwd(), filename)

    generate_mar_file(config=MAR_CONFIG, save_path=EXPORT_PATH)

    MAR_CONFIG.pop(optional_arg)

def test_config_prop_invalid_url():
    """Test mar generation with invalid config.properties url."""
    config_prop_url = "dummy"
    MAR_CONFIG["CONFIG_PROPERTIES"] = "dummy"
    exception_msg = ("Unable to download config properties file using url - {}"
                     .format(config_prop_url))
    with pytest.raises(ValueError, match=exception_msg):
        generate_mar_file(config=MAR_CONFIG, save_path=EXPORT_PATH)
