# Copyright 2022 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import os

import yaml


class FileUtils:

    PROJECT_ROOT = os.path.abspath(
        os.path.join(__file__, *([os.path.pardir] * 5)))
    TEST_DATA = os.path.join(PROJECT_ROOT, "test_data")
    PYTHON_FUNCTIONS = os.path.join(TEST_DATA, "python_functions")
    VALID_PIPELINE_FILES = os.path.join(TEST_DATA, "pipeline_files", "valid")
    ESSENTIAL_PIPELINE_FILES = os.path.join(VALID_PIPELINE_FILES, "essential")
    CRITICAL_PIPELINE_FILES = os.path.join(VALID_PIPELINE_FILES, "critical")
    COMPONENTS = os.path.join(TEST_DATA, "components")

    @classmethod
    def read_yaml_file(cls, filepath) -> tuple:
        """Read the pipeline spec file at the specific file path and parse it
         into a dict and return a tuple of (pipeline_spec, platform_spec) :param
         filepath:

         :return:
         """

        pipeline_specs: dict = None
        platform_specs: dict = None
        with open(filepath, 'r') as file:
            try:
                yaml_data = yaml.safe_load_all(file)
                for data in yaml_data:
                    if 'pipelineInfo' in data.keys():
                        pipeline_specs = data
                    else:
                        platform_specs = data
                return pipeline_specs, platform_specs
            except yaml.YAMLError as ex:
                print(f'Error parsing YAML file: {ex}')
                raise f'Could not load yaml file: {filepath} due to {ex}'
