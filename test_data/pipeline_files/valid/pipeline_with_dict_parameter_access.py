# Copyright 2025 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is# distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Pipeline demonstrating dict parameter access features."""

from typing import Dict

from kfp import compiler
from kfp import dsl


@dsl.component
def print_string(text: str):
    print(text)


@dsl.component
def print_dict(config: dict):
    print(config)


@dsl.component
def use_credentials(username: str, password: str):
    print(f'User: {username}, Pass: {password}')


@dsl.pipeline(name='pipeline-with-dict-parameter-access')
def pipeline_with_dict_parameter_access(
    config: Dict = {
        'app_name': 'TestApp',
        'database': {
            'host': 'localhost',
            'port': '5432',
            'credentials': {
                'username': 'admin',
                'password': 'secret'
            }
        }
    }
):
    """Pipeline demonstrating dict parameter access.
    
    Features demonstrated:
    - Single-level access: config['app_name']
    - Nested access: config['database']['host']
    - Deep nested access: config['database']['credentials']['username']
    - Sub-dict passing: config['database']
    """
    # Single-level access
    print_string(text=config['app_name'])
    
    # Nested access (2 levels)
    print_string(text=config['database']['host'])
    print_string(text=config['database']['port'])
    
    # Deep nested access (3 levels)
    use_credentials(
        username=config['database']['credentials']['username'],
        password=config['database']['credentials']['password'],
    )
    
    # Sub-dict passing
    print_dict(config=config['database'])


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline_with_dict_parameter_access,
        package_path=__file__.replace('.py', '.yaml'))

