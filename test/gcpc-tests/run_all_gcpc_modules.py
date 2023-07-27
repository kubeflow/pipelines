# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Test running all GCPC modules."""
import importlib
import pkgutil
import unittest


def run_all_modules(package_name: str) -> None:
    package = importlib.import_module(package_name)
    for _, module_name, ispkg in pkgutil.walk_packages(package.__path__):
        # use dots to avoid false positives on packages with google in name
        # and train test split packages
        if '.test' in package_name:
            continue
        if ispkg:
            run_all_modules(f'{package_name}.{module_name}')
        else:
            importlib.import_module(f'{package_name}.{module_name}')
        print(f'Successfully ran: {package_name}')


class TestRunAllGCPCModules(unittest.TestCase):

    def test_run_all_modules(self):
        run_all_modules('google_cloud_pipeline_components.preview')
        run_all_modules('google_cloud_pipeline_components.v1')
