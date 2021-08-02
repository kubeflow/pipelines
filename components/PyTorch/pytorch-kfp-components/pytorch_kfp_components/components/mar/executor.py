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
"""Mar Generation Executor Class."""

import os
import shutil
import tempfile
import subprocess
from pathlib import Path
import wget
from pytorch_kfp_components.components.base.base_executor import BaseExecutor
from pytorch_kfp_components.types import standard_component_specs


class Executor(BaseExecutor):
    """Mar Generation Executor Class."""

    def __init__(self):  #pylint: disable=W0235
        """Initializing the executor."""
        super(Executor, self).__init__()  #pylint: disable=R1725

    def get_fn_args(self, input_dict: dict, exec_properties: dict):  #pylint: disable=R0201
        """Extracts the mar config from the input dict and save path from
        exec_properties.

        Args:
            input_dict : A input dict, Eample : mar config dict
            exec_properties : A dict of execution properties for mar saving path
        Returns:
            mar_config : the dict of configuration for mar generation
            mar_save_path : the path for mar file saving
        """

        mar_config = input_dict.get(
            standard_component_specs.MAR_GENERATION_CONFIG
        )
        mar_save_path = exec_properties.get(
            standard_component_specs.MAR_GENERATION_SAVE_PATH
        )
        return mar_config, mar_save_path

    def _validate_mar_config(self, mar_config):  #pylint: disable=R0201
        """Validates the mar configuration properties.

        Args:
            mar_config : a dict of mar configurations.
        Raises:
            ValueError: if the config file or any mandatory arg is empty
        """

        mandatory_args = [
            "MODEL_NAME",
            "SERIALIZED_FILE",
            "MODEL_FILE",
            "HANDLER",
            "VERSION",
            "CONFIG_PROPERTIES",
        ]

        if not mar_config:
            raise ValueError(
                f"Mar config cannot be empty. "
                f"Mandatory arguments are {mandatory_args}"
            )

        missing_list = []
        for key in mandatory_args:
            if key not in mar_config:
                missing_list.append(key)

        if missing_list:
            raise ValueError(
                "Following Mandatory keys are missing in the config file {}".
                format(missing_list)
            )

    def download_config_properties(self, url):  #pylint: disable=R0201
        """Downloads the config.properties.

        Args:
            url : the path to download the config.properties
        """
        if not os.path.exists(url):
            try:
                url = wget.download(url, tempfile.mkdtemp())
            except ValueError as exc:
                raise ValueError(
                    "Unable to download config "
                    "properties file using url - {}".format(url)
                ) from exc

        return url

    def _generate_mar_file(
        self, mar_config: dict, mar_save_path: str, output_dict: dict
    ):
        """Generates the model mar file.

        Args:
            mar_config : mar configuration dict
            mar_save_path : path to save the mar file.
            output_dict : the output dict for saving the mar file.
        Raises:
            Exception :  If archiver command is unable to create mar in case of
                        return code 0.
        """

        self._validate_mar_config(mar_config=mar_config)

        for key, uri in mar_config.items():
            # uri = self._download_dependent_file(key, uri)
            mar_config[key] = uri

        archiver_cmd = (
            "torch-model-archiver --force "
            "--model-name {MODEL_NAME} "
            "--serialized-file {SERIALIZED_FILE} "
            "--model-file {MODEL_FILE} "
            "--handler {HANDLER} "
            "-v {VERSION}".format(
                MODEL_NAME=mar_config["MODEL_NAME"],
                SERIALIZED_FILE=mar_config["SERIALIZED_FILE"],
                MODEL_FILE=mar_config["MODEL_FILE"],
                HANDLER=mar_config["HANDLER"],
                VERSION=mar_config["VERSION"],
            )
        )

        if "EXPORT_PATH" in mar_config:
            export_path = mar_config["EXPORT_PATH"]
            output_dict[standard_component_specs.MAR_GENERATION_SAVE_PATH
                       ] = export_path
            if not os.path.exists(export_path):
                Path(export_path).mkdir(parents=True, exist_ok=True)

            archiver_cmd += " --export-path {EXPORT_PATH}".format(
                EXPORT_PATH=export_path
            )

        if "EXTRA_FILES" in mar_config:
            archiver_cmd += " --extra-files {EXTRA_FILES}".format(
                EXTRA_FILES=mar_config["EXTRA_FILES"]
            )

        if "REQUIREMENTS_FILE" in mar_config:
            archiver_cmd += " -r {REQUIREMENTS_FILE}".format(
                REQUIREMENTS_FILE=mar_config["REQUIREMENTS_FILE"]
            )

        print("Running Archiver cmd: ", archiver_cmd)

        with subprocess.Popen(archiver_cmd, shell=True, stdout=subprocess.PIPE,
                              stderr=subprocess.PIPE) as proc:
            _, err = proc.communicate()
            if err:
                raise ValueError(err)

        # If user has provided the export path
        # By default, torch-model-archiver
        # generates the mar file inside the export path

        # If the user has not provieded the export path
        # mar file will be generated in the current working directory
        # The mar file needs to be moved into mar_save_path

        if "EXPORT_PATH" not in mar_config:
            mar_file_local_path = os.path.join(
                os.getcwd(), "{}.mar".format(mar_config["MODEL_NAME"])
            )
            if not Path(mar_save_path).exists():
                Path(mar_save_path).mkdir(parents=True, exist_ok=True)
            shutil.move(mar_file_local_path, mar_save_path)
            output_dict[standard_component_specs.MAR_GENERATION_SAVE_PATH
                       ] = mar_save_path

        elif mar_config["EXPORT_PATH"] != mar_save_path:
            raise Exception(
                "The export path [{}] needs to be same as mar save path [{}] ".
                format(mar_config["EXPORT_PATH"], mar_save_path)
            )

        print("Saving model file ")
        ## TODO: While separating the mar generation component from trainer # pylint: disable=W0511
        ## Create a separate url for model file
        print(
            f"copying {mar_config['MODEL_FILE']} to {mar_config['EXPORT_PATH']}"
        )
        shutil.copy(mar_config["MODEL_FILE"], mar_config["EXPORT_PATH"])

    def _save_config_properties(
        self, mar_config: dict, mar_save_path: str, output_dict: dict
    ):
        """Saves the config.properties file where the mar file is generated.

        Args :
            mar_config : dict of mar configuration
            mar_save_path : the location to save the config.properties
            output_dict : dict to assign the mar save path.
        """
        print("Downloading config properties")
        if "CONFIG_PROPERTIES" in mar_config:
            config_properties_local_path = self.download_config_properties(
                mar_config["CONFIG_PROPERTIES"]
            )
        else:
            config_properties_local_path = mar_config["CONFIG_PROPERTIES"]

        config_prop_path = os.path.join(mar_save_path, "config.properties")
        if os.path.exists(config_prop_path):
            os.remove(config_prop_path)
        shutil.move(config_properties_local_path, mar_save_path)
        output_dict[standard_component_specs.CONFIG_PROPERTIES_SAVE_PATH
                   ] = mar_save_path

    def Do(self, input_dict: dict, output_dict: dict, exec_properties: dict):
        """Executes the mar generation process.

        Args:
            input_dict : A input dict, including mar config
            output_dict :
            exec_properties : A dict of execution properties for mar generation
        """
        self._log_startup(
            input_dict=input_dict,
            output_dict=output_dict,
            exec_properties=exec_properties,
        )
        mar_config, mar_save_path = self.get_fn_args(
            input_dict=input_dict, exec_properties=exec_properties
        )
        self._validate_mar_config(mar_config=mar_config)

        self._generate_mar_file(
            mar_config=mar_config,
            mar_save_path=mar_save_path,
            output_dict=output_dict,
        )
        self._save_config_properties(
            mar_config=mar_config,
            mar_save_path=mar_save_path,
            output_dict=output_dict,
        )
