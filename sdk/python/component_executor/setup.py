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

import setuptools

NAME = 'kfp-component-executor'
VERSION = '0.0.1'

readme_path = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), 'README.md')

with open(readme_path) as f:
    readme = f.read()

setuptools.setup(
    name=NAME,
    version=VERSION,
    description='Kubeflow Pipelines component execution runtime.',
    long_description=readme,
    author='google',
    author_email='kubeflow-pipelines@google.com',
    url='https://github.com/kubeflow/pipelines',
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    install_requires=['typing-extensions>=3.7.4,<5'],
    python_requires='>=3.7.0',
    license='Apache 2.0',
)
