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
"""Training Executor class."""

import os
from argparse import Namespace
import pytorch_lightning as pl
import torch
from pytorch_kfp_components.components.trainer.generic_executor import (
    GenericExecutor,
)
from pytorch_kfp_components.types import standard_component_specs


class Executor(GenericExecutor):
    """The Training Executor class."""

    def __init__(self):  # pylint:disable=useless-super-delegation
        super().__init__()

    def Do(self, input_dict: dict, output_dict: dict, exec_properties: dict):  #pylint: disable=too-many-locals
        """This function of the Executor invokes the PyTorch Lightning training
        loop.

        Args:
            input_dict : The dictionary of inputs.Example
            : model file, data module file
            output_dict :
            exec_properties : A dict of execution properties
                            including data_module_args,
                             trainer_args, module_file_args

        Returns:
            trainer : The object of PyTorch-Lightning Trainer.

        Raises:
            ValueError : If both of module_file_arfs or trainer_args are empty.
            TypeError : If the type of trainer_args is not dict.
            NotImplementedError : If mandatory args;
                                module_file or data_module_file is empty.
        """
        self._log_startup(
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )

        (
            module_file,
            data_module_file,
            trainer_args,
            module_file_args,
            data_module_args,
        ) = self._get_fn_args(
            input_dict=input_dict,
            output_dict=output_dict,
            execution_properties=exec_properties,
        )

        (
            model_class,
            data_module_class,
        ) = self.derive_model_and_data_module_class(
            module_file=module_file, data_module_file=data_module_file
        )
        if not data_module_class :
            raise NotImplementedError(
                "Data module class is mandatory. "
                "User defined training module is yet to be supported."
            )
        if data_module_class:
            data_module = data_module_class(
                **data_module_args if data_module_args else {}
            )
            data_module.prepare_data()
            data_module.setup(stage="fit")
            model = model_class(**module_file_args if module_file_args else {})

            if (not module_file_args) and (not trainer_args):
                raise ValueError("Module file & trainer args can't be empty")

            if not isinstance(trainer_args, dict):
                raise TypeError("trainer_args must be a dict")

            module_file_args.update(trainer_args)
            parser = Namespace(**module_file_args)
            trainer = pl.Trainer.from_argparse_args(parser)

            trainer.fit(model, data_module)  #pylint: disable=no-member
            trainer.test(datamodule=data_module, model=model)  #pylint: disable=no-member

            if "checkpoint_dir" in module_file_args:
                model_save_path = module_file_args["checkpoint_dir"]
            else:
                model_save_path = "/tmp"

            if "model_name" in module_file_args:
                model_name = module_file_args["model_name"]
            else:
                model_name = "model_state_dict.pth"

            model_save_path = os.path.join(model_save_path, model_name)
            if trainer.global_rank == 0:
                print("Saving model to {}".format(model_save_path))
                torch.save(model.state_dict(), model_save_path)

            output_dict[standard_component_specs.TRAINER_MODEL_SAVE_PATH
                       ] = model_save_path
            output_dict[standard_component_specs.PTL_TRAINER_OBJ] = trainer
