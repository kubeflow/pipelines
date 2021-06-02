#!/usr/bin/env/python3
# Copyright (c) Facebook, Inc. and its affiliates.
# All rights reserved.
#
# This source code is licensed under the BSD-style license found in the
# LICENSE file in the root directory of this source tree.
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
