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
"""Training Component class."""

from typing import Optional, Dict
from pytorch_kfp_components.components.trainer.executor import Executor
from pytorch_kfp_components.components.base.base_component import BaseComponent
from pytorch_kfp_components.types import standard_component_specs


class Trainer(BaseComponent):  #pylint: disable=too-few-public-methods
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
