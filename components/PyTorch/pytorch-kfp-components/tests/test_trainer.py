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
"""Unit tests for trainer component."""
import os
import shutil
import sys
import tempfile

import pytest
import pytorch_lightning

from pytorch_kfp_components.components.trainer.component import Trainer

dirname, filename = os.path.split(os.path.abspath(__file__))
IRIS_DIR = os.path.join(dirname, "iris")
sys.path.insert(0, IRIS_DIR)

MODULE_FILE_ARGS = {"lr": 0.1}
TRAINER_ARGS = {"max_epochs": 5}
DATA_MODULE_ARGS = {"num_workers": 2}

# pylint:disable=redefined-outer-name


@pytest.fixture(scope="class")
def trainer_params():
    trainer_params = {
        "module_file": "iris_classification.py",
        "data_module_file": "iris_data_module.py",
        "module_file_args": MODULE_FILE_ARGS,
        "data_module_args": DATA_MODULE_ARGS,
        "trainer_args": TRAINER_ARGS,
    }
    return trainer_params


MANDATORY_ARGS = [
    "module_file",
    "data_module_file",
]
OPTIONAL_ARGS = ["module_file_args", "data_module_args", "trainer_args"]

DEFAULT_MODEL_NAME = "model_state_dict.pth"
DEFAULT_SAVE_PATH = f"/tmp/{DEFAULT_MODEL_NAME}"


def invoke_training(trainer_params):  # pylint: disable=W0621
    """This function invokes the training process."""
    trainer = Trainer(
        module_file=trainer_params["module_file"],
        data_module_file=trainer_params["data_module_file"],
        module_file_args=trainer_params["module_file_args"],
        trainer_args=trainer_params["trainer_args"],
        data_module_args=trainer_params["data_module_args"],
    )
    return trainer


@pytest.mark.parametrize("mandatory_key", MANDATORY_ARGS)
def test_mandatory_keys_type_check(trainer_params, mandatory_key):
    """Tests the uncexpected 'type' of mandatory args.
    Args:
        mandatory_key : mandatory arguments for inivoking training
    """
    test_input = ["input_path"]
    trainer_params[mandatory_key] = test_input
    expected_exception_msg = (
        f"{mandatory_key} must be of type <class 'str'> "
        f"but received as {type(test_input)}"
    )
    with pytest.raises(TypeError, match=expected_exception_msg):
        invoke_training(trainer_params=trainer_params)


@pytest.mark.parametrize("optional_key", OPTIONAL_ARGS)
def test_optional_keys_type_check(trainer_params, optional_key):
    """Tests the unexpected 'type' of optional args.
    Args:
        optional_key: optional arguments for invoking training
    """
    test_input = "test_input"
    trainer_params[optional_key] = test_input
    expected_exception_msg = (
        f"{optional_key} must be of type <class 'dict'> "
        f"but received as {type(test_input)}"
    )
    with pytest.raises(TypeError, match=expected_exception_msg):
        invoke_training(trainer_params=trainer_params)


@pytest.mark.parametrize("input_key", MANDATORY_ARGS + ["module_file_args"])
def test_mandatory_params(trainer_params, input_key):
    """Test for empty mandatory arguments.
    Args:
        input_key: name of the mandatory arg for training
    """
    trainer_params[input_key] = None
    expected_exception_msg = (
        f"{input_key} is not optional. "
        f"Received value: {trainer_params[input_key]}"
    )
    with pytest.raises(ValueError, match=expected_exception_msg):
        invoke_training(trainer_params=trainer_params)


def test_data_module_args_optional(trainer_params):
    """Test for empty optional argument : data module args"""
    trainer_params["data_module_args"] = None
    invoke_training(trainer_params=trainer_params)
    assert os.path.exists(DEFAULT_SAVE_PATH)
    os.remove(DEFAULT_SAVE_PATH)


def test_trainer_args_none(trainer_params):
    """Test for empty trainer specific arguments."""
    trainer_params["trainer_args"] = None
    expected_exception_msg = r"trainer_args must be a dict"
    with pytest.raises(TypeError, match=expected_exception_msg):
        invoke_training(trainer_params=trainer_params)


def test_training_success(trainer_params):
    """Test the training success case with all required args."""
    trainer = invoke_training(trainer_params=trainer_params)
    assert os.path.exists(DEFAULT_SAVE_PATH)
    os.remove(DEFAULT_SAVE_PATH)
    assert hasattr(trainer, "ptl_trainer")
    assert isinstance(
        trainer.ptl_trainer, pytorch_lightning.trainer.trainer.Trainer
    )


def test_training_success_with_custom_model_name(trainer_params):
    """Test for successful training with custom model name."""
    tmp_dir = tempfile.mkdtemp()
    trainer_params["module_file_args"]["checkpoint_dir"] = tmp_dir
    trainer_params["module_file_args"]["model_name"] = "iris.pth"
    invoke_training(trainer_params=trainer_params)
    assert "iris.pth" in os.listdir(tmp_dir)
    shutil.rmtree(tmp_dir)
    trainer_params["module_file_args"].pop("checkpoint_dir")
    trainer_params["module_file_args"].pop("model_name")


def test_training_failure_with_empty_module_file_args(trainer_params):
    """Test for successful training with empty module file args."""
    trainer_params["module_file_args"] = {}
    exception_msg = "module_file_args is not optional. Received value: {}"
    with pytest.raises(ValueError, match=exception_msg):
        invoke_training(trainer_params=trainer_params)


def test_training_success_with_empty_trainer_args(trainer_params):
    """Test for successful training with empty trainer args."""
    tmp_dir = tempfile.mkdtemp()
    trainer_params["module_file_args"]["max_epochs"] = 5
    trainer_params["module_file_args"]["checkpoint_dir"] = tmp_dir
    trainer_params["trainer_args"] = {}
    invoke_training(trainer_params=trainer_params)
    assert DEFAULT_MODEL_NAME in os.listdir(tmp_dir)
    shutil.rmtree(tmp_dir)


def test_training_success_with_empty_data_module_args(trainer_params):
    """Test for successful training with empty data module args."""
    tmp_dir = tempfile.mkdtemp()
    trainer_params["module_file_args"]["checkpoint_dir"] = tmp_dir
    trainer_params["data_module_args"] = None
    invoke_training(trainer_params=trainer_params)

    assert DEFAULT_MODEL_NAME in os.listdir(tmp_dir)
    shutil.rmtree(tmp_dir)


#
def test_trainer_output(trainer_params):
    """Test for successful training with proper saving of training output."""
    tmp_dir = tempfile.mkdtemp()
    trainer_params["module_file_args"]["checkpoint_dir"] = tmp_dir
    trainer = invoke_training(trainer_params=trainer_params)

    assert hasattr(trainer, "output_dict")
    assert trainer.output_dict is not None
    assert trainer.output_dict["model_save_path"] == os.path.join(
        tmp_dir, DEFAULT_MODEL_NAME
    )
    assert isinstance(
        trainer.output_dict["ptl_trainer"],
        pytorch_lightning.trainer.trainer.Trainer
    )
