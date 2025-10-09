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
import re

from sdk.python import kfp


class ComparisonUtils:

    @classmethod
    def compare_pipeline_spec_dicts(cls, actual: dict, expected: dict,
                                    **kwargs):
        """Compare two pipeline/platform specs :param actual: Pipeline Spec
        that you want to compare :param expected: Pipeline Spec that is the
        source of truth :param kwargs: options are: name (pipeline name),
        display_name (pipeline display name), runtime_params (pipeline runtime
        params)"""
        if expected is None:
            assert actual is None, "Actual is not None when its expected to be None"
        else:
            for key, value in expected.items():
                if type(value) == dict:
                    # Override Pipeline Name and Display Name if those were overridden during compilation
                    if key == 'pipelineInfo':
                        value['name'] = kwargs['name']

                    # Override Run Time Params in the expected object if runtime params were overridden when compiling pipeline
                    if 'runtime_params' in kwargs:
                        if kwargs['runtime_params'] is not None:
                            if key == 'root':
                                for param_key, param_value in value[
                                        'inputDefinitions']['parameters'].items(
                                        ):
                                    if param_key in kwargs[
                                            'runtime_params'].keys():
                                        value['inputDefinitions']['parameters'][
                                            param_key]['defaultValue'] = kwargs[
                                                'runtime_params'][param_key]

                    cls.compare_pipeline_spec_dicts(actual[key], value,
                                                    **kwargs)
                else:
                    # Override SDK Version to match the current version
                    if key == 'sdkVersion':
                        value = f'kfp-{kfp.__version__}'
                    # Override SDK Version in the args as well to match the current version
                    if key == 'command':
                        for index, command in enumerate(value):
                            if re.search("kfp==[0-9].[0-9]+.[0-9]+",
                                         command) is not None:
                                value[index] = re.sub(
                                    "kfp==[0-9].[0-9]+.[0-9]+",
                                    f"kfp=={kfp.__version__}", command)
                    assert value == actual[
                        key], f'Value for "{key}" is not the same'
