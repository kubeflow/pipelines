# Copyright 2023 The Kubeflow Authors
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
import re

import setuptools


def find_version(*file_path_parts: str) -> str:
    """Get version from a file that defines a __version__ variable."""

    file_path = os.path.join(os.path.dirname(__file__), *file_path_parts)
    with open(file_path, 'r') as f:
        version_file_text = f.read()

    version_match = re.search(
        r"^__version__ = ['\"]([^'\"]*)['\"]",
        version_file_text,
        re.M,
    )
    if version_match:
        return version_match.group(1)

    raise RuntimeError(f'Unable to find version string in file: {file_path}.')


setuptools.setup(
    name='kfp-dsl',
    version=find_version(
        os.path.dirname(os.path.dirname(__file__)), 'kfp', '__init__.py'),
    description='A KFP SDK subpackage containing the DSL and runtime code.',
    author='google',
    author_email='kubeflow-pipelines@google.com',
    url='https://github.com/kubeflow/pipelines',
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    python_requires='>=3.7.0',
    install_requires=['typing-extensions>=3.7.4,<5; python_version<"3.9"'],
    include_package_data=True,
    license='Apache 2.0',
)
