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
"""Mar Generation Component Class."""

from pytorch_kfp_components.components.mar.executor import Executor
from pytorch_kfp_components.types import standard_component_specs
from pytorch_kfp_components.components.base.base_component import BaseComponent


class MarGeneration(BaseComponent):  #pylint: disable=R0903
    """Mar generation class."""
    def __init__(self, mar_config: dict, mar_save_path: str = None):
        """Initializes the Mar Generation class.

        Args:
            mar_config: mar configuration dict (type:dict)
            mar_save_path : the path for saving the mar file (type:str)
        """
        super(MarGeneration, self).__init__()  #pylint: disable=R1725
        input_dict = {
            standard_component_specs.MAR_GENERATION_CONFIG: mar_config,
        }

        output_dict = {}

        exec_properties = {
            standard_component_specs.MAR_GENERATION_SAVE_PATH: mar_save_path
        }

        spec = standard_component_specs.MarGenerationSpec()
        self._validate_spec(
            spec=spec,
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )

        Executor().Do(
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )
        self.output_dict = output_dict
