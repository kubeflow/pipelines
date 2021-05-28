import os
import shutil
import sys
import tempfile
from copy import deepcopy

import pytest
import pytorch_lightning

from pytorch_kfp_components.components.trainer.component import Trainer

dirname, filename = os.path.split(os.path.abspath(__file__))
print("*"*100,dirname)
print("+"*100,filename)

IRIS_DIR = os.path.join(dirname,"iris")
sys.path.insert(0, IRIS_DIR)

MODULE_FILE_ARGS = {"lr": 0.1}
TRAINER_ARGS = {"max_epochs": 5}
DATA_MODULE_ARGS = {"num_workers": 2}
trainer_params = {
    "module_file": "iris_classification.py",
    "data_module_file": "iris_data_module.py",
    "module_file_args": MODULE_FILE_ARGS,
    "data_module_args": DATA_MODULE_ARGS,
    "trainer_args": TRAINER_ARGS,
}

MANDATORY_ARGS = [
    "module_file",
    "data_module_file",
]

DEFAULT_MODEL_NAME = "model_state_dict.pth"
DEFAULT_SAVE_PATH = f"/tmp/{DEFAULT_MODEL_NAME}"


def invoke_training(trainer_params):
    trainer = Trainer(
        module_file=trainer_params["module_file"],
        data_module_file=trainer_params["data_module_file"],
        module_file_args=trainer_params["module_file_args"],
        trainer_args=trainer_params["trainer_args"],
        data_module_args=trainer_params["data_module_args"],
    )
    return trainer


@pytest.mark.parametrize("mandatory_key", MANDATORY_ARGS)
def test_mandatory_parameters_missing(mandatory_key):
    mandatory_trainer_dict = deepcopy(trainer_params)
    mandatory_trainer_dict[mandatory_key] = None
    expected_exception_msg = f"{mandatory_key} cannot be None"
    with pytest.raises(ValueError, match=expected_exception_msg):
        invoke_training(trainer_params=mandatory_trainer_dict)


@pytest.mark.parametrize("key", ["module_file", "data_module_file"])
def test_invalid_module_file(key):
    invalid_module_dict = deepcopy(trainer_params)
    invalid_module_dict[key] = "iris.py"
    expected_exception_msg = f"Unable to load {key} - iris.py"
    with pytest.raises(ValueError, match=expected_exception_msg):
        invoke_training(trainer_params=invalid_module_dict)


def test_module_file_and_trainer_args_empty():
    empty_args_dict = deepcopy(trainer_params)
    empty_args_dict["module_file_args"] = None
    empty_args_dict["trainer_args"] = None

    expected_exception_msg = "Both module file args and trainer args cannot be empty"
    with pytest.raises(ValueError, match=expected_exception_msg):
        invoke_training(trainer_params=empty_args_dict)


def test_training_success():
    trainer = invoke_training(trainer_params=trainer_params)
    assert os.path.exists(DEFAULT_SAVE_PATH)
    os.remove(DEFAULT_SAVE_PATH)
    assert hasattr(trainer, "ptl_trainer")
    assert isinstance(trainer.ptl_trainer, pytorch_lightning.trainer.trainer.Trainer)


def test_training_success_with_custom_model_name():
    tmp_dir = tempfile.mkdtemp()
    custom_model_name_dict = deepcopy(trainer_params)
    custom_model_name_dict["module_file_args"]["checkpoint_dir"] = tmp_dir
    custom_model_name_dict["module_file_args"]["model_name"] = "iris.pth"
    invoke_training(trainer_params=custom_model_name_dict)
    assert "iris.pth" in os.listdir(tmp_dir)
    shutil.rmtree(tmp_dir)


def test_training_success_with_empty_module_file_args():
    empty_args_dict = deepcopy(trainer_params)
    empty_args_dict["module_file_args"] = {}
    invoke_training(trainer_params=trainer_params)


def test_training_success_with_empty_trainer_args():
    tmp_dir = tempfile.mkdtemp()
    empty_args_dict = deepcopy(trainer_params)
    empty_args_dict["module_file_args"]["max_epochs"] = 5
    empty_args_dict["module_file_args"]["checkpoint_dir"] = tmp_dir
    empty_args_dict["trainer_args"] = {}
    invoke_training(trainer_params=empty_args_dict)
    assert DEFAULT_MODEL_NAME in os.listdir(tmp_dir)
    shutil.rmtree(tmp_dir)


def test_training_success_with_empty_data_module_args():
    tmp_dir = tempfile.mkdtemp()
    tmp_trainer_parms = deepcopy(trainer_params)
    tmp_trainer_parms["module_file_args"]["checkpoint_dir"] = tmp_dir
    tmp_trainer_parms["data_module_args"] = None
    invoke_training(trainer_params=tmp_trainer_parms)

    assert DEFAULT_MODEL_NAME in os.listdir(tmp_dir)
    shutil.rmtree(tmp_dir)


def test_empty_args_type_error():
    empty_args_dict = deepcopy(trainer_params)
    empty_args_dict["trainer_args"] = None
    with pytest.raises(TypeError, match="trainer_args must be a dict"):
        invoke_training(trainer_params=empty_args_dict)
