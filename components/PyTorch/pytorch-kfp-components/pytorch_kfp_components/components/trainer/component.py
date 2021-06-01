#!/usr/bin/env/python3
# Copyright (c) Facebook, Inc. and its affiliates.
# All rights reserved.
# This source code is licensed under the BSD-style license found in the
# LICENSE file in the root directory of this source tree.
"""Training Component class."""

from typing import Optional, Dict
from pytorch_kfp_components.components.trainer.executor import Executor
from pytorch_kfp_components.components.base.base_component import BaseComponent
from pytorch_kfp_components.types import standard_component_specs


class Trainer(BaseComponent):
    """Initializes the Trainer class."""

    def __init__(  # pylint: disable=R0913
        self,
        module_file: Optional = None,
        data_module_file: Optional = None,
        data_module_args: Optional[Dict] = None,
        module_file_args: Optional[Dict] = None,
        trainer_args: Optional[Dict] = None,
    ):
        """Initializes the PyTorch Lightning training process.

        Args:
            module_file : Inherit the model class for training.
            data_module_file : From this the data module class is inherited.
            data_module_args : The arguments of the data module.
            module_file_args : The arguments of the model class.
            trainer_args : arguments specific to the PTL trainer.

        Raises:
            NotImplementedError : If mandatory args;
            module_file or data_module_file is empty.
        """

        super(Trainer, self).__init__()  # pylint: disable=R1725
        input_dict = {
            standard_component_specs.TRAINER_MODULE_FILE: module_file,
            standard_component_specs.TRAINER_DATA_MODULE_FILE: data_module_file,
        }

        output_dict = {}

        exec_properties = {
            standard_component_specs.TRAINER_DATA_MODULE_ARGS: data_module_args,
            standard_component_specs.TRAINER_MODULE_ARGS: module_file_args,
            standard_component_specs.PTL_TRAINER_ARGS: trainer_args,
        }

        spec = standard_component_specs.TrainerSpec()
        self._validate_spec(
            spec=spec,
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )

        if module_file and data_module_file:
            # Both module file and data module file are present

            Executor().Do(
                input_dict=input_dict,
                output_dict=output_dict,
                exec_properties=exec_properties,
            )

            self.ptl_trainer = output_dict.get(
                standard_component_specs.PTL_TRAINER_OBJ, "None"
            )
            self.output_dict = output_dict
        else:
            raise NotImplementedError(
                "Module file and Datamodule file are mandatory. "
                "Custom training methods are yet to be implemented"
            )
