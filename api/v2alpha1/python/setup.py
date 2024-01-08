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

NAME = 'kfp-pipeline-spec'
VERSION = '0.3.0'

setuptools.setup(
    name=NAME,
    version=VERSION,
    description='Kubeflow Pipelines pipeline spec',
    author='google',
    author_email='kubeflow-pipelines@google.com',
    url='https://github.com/kubeflow/pipelines',
    packages=setuptools.find_namespace_packages(include=['kfp.*']),
    python_requires='>=3.7.0,<3.13.0',
    install_requires=['protobuf>=4.21.1,<5'],
    include_package_data=True,
    license='Apache 2.0',
)
