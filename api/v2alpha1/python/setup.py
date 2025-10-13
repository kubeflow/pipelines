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

import setuptools
import os
from typing import List

NAME = 'kfp-pipeline-spec'
VERSION = '2.14.3'

def get_requirements(requirements_file: str) -> List[str]:
    """Read requirements from requirements.in."""

    file_path = os.path.join(os.path.dirname(__file__), requirements_file)
    with open(file_path, 'r') as f:
        lines = f.readlines()
    lines = [line.strip() for line in lines]
    lines = [line for line in lines if not line.startswith('#') and line]
    return lines

setuptools.setup(
    name=NAME,
    version=VERSION,
    description='Kubeflow Pipelines pipeline spec',
    author='google',
    author_email='kubeflow-pipelines@google.com',
    url='https://github.com/kubeflow/pipelines',
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    python_requires='>=3.9.0',
    install_requires=get_requirements('requirements.in'),
    include_package_data=True,
    license='Apache 2.0',
)
